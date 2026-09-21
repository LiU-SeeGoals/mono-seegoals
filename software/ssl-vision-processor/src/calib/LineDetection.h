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
#pragma once

#include "Resources.h"


typedef std::pair<cv::Vec2f, cv::Vec2f> CVLine;
typedef std::vector<std::pair<cv::Vec2f, cv::Vec2f>> CVLines;

/* Estimates the half line width without camera model from the field size and camera amount */
int halfLineWidthEstimation(const Resources& r, const cv::Mat& img);

/* Finds line points by detecting ridges */
void thresholdImage(const Resources& r, const cv::Mat& gray, int halfLineWidth, cv::Mat& thresholded);

/* Groups neighbouring line segments together */
std::vector<CVLines> groupLineSegments(const Resources& r, CVLines& segments);

/* Merges a list of grouped line segments to a list of lines */
CVLines mergeLineSegments(const std::vector<CVLines>& compoundLines);

/* Determines the intersection point of two lines */
cv::Vec2f lineLineIntersection(const CVLine& a, const CVLine& b);

/* Finds all intersections of the given line list inside or next to the image. */
std::vector<Eigen::Vector2f> lineIntersections(const CVLines& lines, int width, int height, double maxIntersectionDistance);

/* Determines the clockwise quadrilateral with the largest area for the given point set */
std::list<Eigen::Vector2f> findOuterEdges(const std::vector<cv::Vec2f>& intersections);
