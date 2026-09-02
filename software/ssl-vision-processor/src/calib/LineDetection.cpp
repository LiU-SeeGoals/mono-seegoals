/*
     Copyright 2024 Felix Weinmann

     Licensed under the Apache License, Version 2.0 (the "License");
     you may not use this file except in compliance with the License.
     You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

     Unless required by applicable law or agreed to in writing, software
     distributed under the License is distributed on an "AS IS" BASIS,
     WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
     See the License for the specific language governing permissions and
     limitations under the License.
 */
#include "LineDetection.h"
#include "GeomModel.h"

int halfLineWidthEstimation(const Resources& r, const cv::Mat& img) {
	Eigen::Vector2f min;
	Eigen::Vector2f max;
	visibleFieldExtent(r, true, min, max);

	Eigen::Vector2f extent = max - min;
	Eigen::Vector2f camera(img.cols, img.rows);

	// Assume large field extent side is large camera side
	if(extent[0] < extent[1])
		std::swap(extent[0], extent[1]);
	if(camera[0] < camera[1])
		std::swap(camera[0], camera[1]);

	Eigen::Vector2f ratio = camera.array() / extent.array();

	return std::ceil(std::max(ratio[0], ratio[1]) * (float)r.socket->getGeometry().field().line_thickness()/2.0f);
}

static inline bool threshold(const Resources& r, int value, int neg, int pos) {
	return value - neg > r.fieldLineThreshold && value - pos > r.fieldLineThreshold;
}

void thresholdImage(const Resources& r, const cv::Mat& gray, const int halfLineWidth, cv::Mat& thresholded) {
	for (int y = halfLineWidth; y < gray.rows - halfLineWidth; y++) {
		for (int x = halfLineWidth; x < gray.cols - halfLineWidth; x++) {
			uint8_t value = gray.at<uint8_t>(y, x);
			thresholded.at<uint8_t>(y, x) = (
					threshold(r, value, gray.at<uint8_t>(y, x - halfLineWidth), gray.at<uint8_t>(y, x + halfLineWidth)) ||
					threshold(r, value, gray.at<uint8_t>(y - halfLineWidth, x), gray.at<uint8_t>(y + halfLineWidth, x))
			) ? 255 : 0;
		}
	}
}

std::vector<CVLines> groupLineSegments(const Resources& r, CVLines& segments) {
	std::vector<CVLines> compoundLines;
	while(!segments.empty()) {
		CVLines compound;
		compound.push_back(segments.front());
		segments.erase(segments.cbegin());

		for(unsigned int i = 0; i < compound.size(); i++) {
			const auto& root = compound[i];
			cv::Vec2f v1 = root.second - root.first;

			auto lit = segments.cbegin();
			while(lit != segments.cend()) {
				CVLine l = *lit;
				cv::Vec2f v2 = l.second - l.first;
				if(
						abs(acosf(abs(v2.dot(v1) / (sqrtf(v1.dot(v1)) * sqrtf(v2.dot(v2)))))) <= r.maxLineSegmentAngle &&
						// https://en.wikipedia.org/wiki/Distance_from_a_point_to_a_line
						std::min(abs(v1[0]*(l.first[1] - root.first[1]) - (l.first[0] - root.first[0])*v1[1]) / sqrtf(v1.dot(v1)), abs(v1[0]*(l.second[1] - root.first[1]) - (l.second[0] - root.first[0])*v1[1]) / sqrtf(v1.dot(v1))) <= r.maxLineSegmentOffset &&
						(dist(root.first, l.first) <= 200 || dist(root.second, l.first) <= 200 || dist(root.first, l.second) <= 200 || dist(root.second, l.second) <= 200)
						) {
					compound.push_back(l);
					lit = segments.erase(lit);
				} else {
					lit++;
				}
			}
		}

		//Sort according to line length
		std::sort(compound.begin(), compound.end(), [](const CVLine& v1, const CVLine& v2) { return dist(v1.first, v1.second) > dist(v2.first, v2.second); });
		compoundLines.push_back(compound);
	}
	return compoundLines;
}

CVLines mergeLineSegments(const std::vector<CVLines>& compoundLines) {
	CVLines mergedLines;
	for(const auto& compound : compoundLines) {
		cv::Vec2f a = compound.front().first;
		cv::Vec2f b = compound.front().second;
		for(unsigned int i = 1; i < compound.size(); i++) {
			const auto& v = compound[i];

			const cv::Vec2f& c = v.first;
			const cv::Vec2f& d = v.second;

			cv::Vec2f max1 = a;
			cv::Vec2f max2 = b;
			float maxd = dist(a, b);

			if(dist(a, c) > maxd) {
				max1 = a;
				max2 = c;
				maxd = dist(a, c);
			}
			if(dist(a, d) > maxd) {
				max1 = a;
				max2 = d;
				maxd = dist(a, d);
			}
			if(dist(c, b) > maxd) {
				max1 = c;
				max2 = b;
				maxd = dist(c, b);
			}
			if(dist(d, b) > maxd) {
				max1 = d;
				max2 = b;
				maxd = dist(d, b);
			}
			if(dist(c, d) > maxd) {
				max1 = c;
				max2 = d;
				maxd = dist(c, d);
			}

			a = max1;
			b = max2;
		}
		mergedLines.emplace_back(a, b);
	}
	return mergedLines;
}

// Adapted from https://stackoverflow.com/a/7448287 by Andrey Kamaev CC BY-SA 3.0
cv::Vec2f lineLineIntersection(const CVLine& a, const CVLine& b) {
	cv::Vec2f x = b.first - a.first;
	cv::Vec2f da = a.second - a.first;
	cv::Vec2f db = b.second - b.first;

	float cross = da[0]*db[1] - da[1]*db[0];
	if (abs(cross) < 1e-8)
		return {INFINITY, INFINITY};

	double t1 = (x[0] * db[1] - x[1] * db[0]) / cross;
	return a.first + da * t1;
}

std::vector<Eigen::Vector2f> lineIntersections(const CVLines& lines, const int width, const int height, const double maxIntersectionDistance) {
	std::vector<Eigen::Vector2f> intersections;

	float minX = -width*maxIntersectionDistance;
	float minY = -height*maxIntersectionDistance;
	float maxX = width + width*maxIntersectionDistance;
	float maxY = height + height*maxIntersectionDistance;
	auto ita = lines.cbegin();
	while(ita != lines.cend()) {
		auto itb = ita+1;
		while(itb != lines.cend()) {
			cv::Vec2f c = lineLineIntersection(*ita, *itb);
			if(c[0] >= minX && c[1] >= minY && c[0] < maxX && c[1] < maxY)
				intersections.push_back(cv2eigen(c));

			itb++;
		}

		ita++;
	}

	return intersections;
}

static bool inSegment(const cv::Vec2f& a, const cv::Vec2f& b, const cv::Vec2f& point) {
	return point[0] > std::min(a[0], b[0]) && point[1] > std::min(a[1], b[1]) && point[0] < std::max(a[0], b[0]) && point[1] < std::max(a[1], b[1]);
}

std::list<Eigen::Vector2f> findOuterEdges(const std::vector<cv::Vec2f>& intersections) {
	std::list<Eigen::Vector2f> edges;
	float maxArea = 0.0f;
	for(const cv::Vec2f& a : intersections) {
		for(const cv::Vec2f& b : intersections) {
			for(const cv::Vec2f& c : intersections) {
				for(const cv::Vec2f& d : intersections) {
					if(a == b || a == c || a == d || b == c || b == d || c == d)
						continue;

					cv::Vec2f center = lineLineIntersection({a, c}, {b, d});
					if(!inSegment(a, c, center) || !inSegment(b, d, center)) // Check if ABCD is convex quadrilateral
						continue;

					//https://en.wikipedia.org/wiki/Quadrilateral#Vector_formulas
					cv::Vec2f ac = c - a;
					cv::Vec2f bd = d - b;
					float area = 0.5f * abs(ac[0] * bd[1] - bd[0] * ac[1]);
					if(area < maxArea)
						continue;

					edges.clear();
					edges.push_back(cv2eigen(a));
					edges.push_back(cv2eigen(b));
					edges.push_back(cv2eigen(c));
					edges.push_back(cv2eigen(d));
					maxArea = area;
				}
			}
		}
	}
	return edges;
}
