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

float dist(const cv::Vec2f& v1, const cv::Vec2f& v2);
void visibleFieldExtent(const Resources &r, bool withBoundary, Eigen::Vector2f &min, Eigen::Vector2f &max);
Eigen::Vector2f cv2eigen(const cv::Vec2f& v);

std::vector<Eigen::Vector2f> getLinePixels(const cv::Mat& thresholded);
int modelError(const Resources& r, const CameraModel& model, const std::vector<Eigen::Vector2f>& linePixels);

void geometryCalibration(const Resources& r, const CLImage& img);
