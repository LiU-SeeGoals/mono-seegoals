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

#include <atomic>
#include <condition_variable>
#include <mutex>
#include <thread>

#include <opencv2/videoio.hpp>

#include "cameradriver.h"

class OpenCVDriver : public CameraDriver {
public:
	explicit OpenCVDriver(const CameraConfig& config);
	~OpenCVDriver() override;

	std::shared_ptr<RawImage> readImage() override;

	const PixelFormat format() override;

	double expectedFrametime() override;

	double getTime() override;
	double getCaptureTime() override;

private:
	void captureLiveFrames();
	std::shared_ptr<RawImage> readLatestImage();

	cv::VideoCapture capture;
	std::shared_ptr<RawImage> image = nullptr;
	std::string name;
	const bool liveSource;
	double frameTime = 1.0 / 30.0;

	// Only the capture thread accesses VideoCapture once live capture starts.
	// The pending frame is replaced, never queued behind an older frame.
	std::mutex frameMutex;
	std::condition_variable frameReady;
	cv::Mat latestFrame;
	double latestFrameTime = 0.0; // Guarded by frameMutex.
	double readFrameTime = 0.0; // Processing thread only.
	bool captureFinished = false; // Guarded by frameMutex.
	std::atomic<bool> stopCapture{false};
	std::thread captureThread;
};
