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
#include "opencvdriver.h"
#include "log.h"

#include <cmath>
#include <utility>
#include <vector>

OpenCVDriver::OpenCVDriver(const CameraConfig& config): name(config.path), liveSource(config.path.find("://") != std::string::npos || config.path.starts_with("/dev/")) {
	std::vector<int> parameters{cv::CAP_PROP_HW_ACCELERATION, cv::VIDEO_ACCELERATION_ANY};
	if(config.path.find("://") != std::string::npos) {
		// FFmpeg/GStreamer read timeouts must be supplied when opening the stream.
		// Allow the capture thread to exit if a network camera stops responding.
		parameters.insert(parameters.end(), {cv::CAP_PROP_READ_TIMEOUT_MSEC, 3000});
	}
	capture.open(config.path, cv::CAP_ANY, parameters);
	if(!capture.isOpened())
		FATAL("Could not open OpenCV input. Check the camera URL, credentials and stream settings.");

	std::replace(name.begin(), name.end(), '/', '_');

	// Use compressed data stream to unlock the highest resolution - framerate combination on USB2 cameras
	capture.set(cv::CAP_PROP_FOURCC, cv::VideoWriter::fourcc('M', 'J', 'P', 'G'));

	if(config.autoResolution()) {
		capture.set(cv::CAP_PROP_FRAME_WIDTH, INT_MAX);
		capture.set(cv::CAP_PROP_FRAME_HEIGHT, INT_MAX);
	} else {
		capture.set(cv::CAP_PROP_FRAME_WIDTH, config.width);
		capture.set(cv::CAP_PROP_FRAME_HEIGHT, config.height);
	}

	if(config.autoExposure()) {
		capture.set(cv::CAP_PROP_AUTO_EXPOSURE, 1.0);
	} else {
		capture.set(cv::CAP_PROP_AUTO_EXPOSURE, 0.0);
		capture.set(cv::CAP_PROP_EXPOSURE, config.exposure * 1000.0);
	}

	if(!config.autoGain()) {
		capture.set(cv::CAP_PROP_GAIN, config.gain);
	}

	if(config.autoGamma()) {
		capture.set(cv::CAP_PROP_GAMMA, config.gamma);
	}

	if(config.whiteBalanceType != WhiteBalanceType_Manual) {
		capture.set(cv::CAP_PROP_AUTO_WB, 1.0);
	} else {
		capture.set(cv::CAP_PROP_AUTO_WB, 0.0);
		capture.set(cv::CAP_PROP_WHITE_BALANCE_BLUE_U, config.whiteBalanceBlue);
		capture.set(cv::CAP_PROP_WHITE_BALANCE_RED_V, config.whiteBalanceRed);
	}

	// Cache this before the worker starts; VideoCapture must not be queried
	// from the processing thread while the capture thread is reading it.
	const double fps = capture.get(cv::CAP_PROP_FPS);
	if(std::isfinite(fps) && fps > 0.0)
		frameTime = 1.0 / fps;

	if(liveSource)
		captureThread = std::thread(&OpenCVDriver::captureLiveFrames, this);
}

OpenCVDriver::~OpenCVDriver() {
	stopCapture = true;
	if(captureThread.joinable())
		captureThread.join();
}

void OpenCVDriver::captureLiveFrames() {
	try {
		cv::Mat frame;
		while(!stopCapture) {
			if(!capture.read(frame) || frame.empty()) {
				if(!stopCapture)
					WARN("Camera stream stopped or frame decoding failed. Restart vision after changing camera settings.");
				break;
			}

			{
				std::lock_guard<std::mutex> lock(frameMutex);
				if(stopCapture)
					break;
				// Reuse the replaced buffer for the next read. A consumed frame
				// has been moved out and cannot be overwritten by the worker.
				std::swap(latestFrame, frame);
			}
			frameReady.notify_one();
		}
	} catch(const cv::Exception&) {
		if(!stopCapture)
			WARN("Camera stream decoding failed. Check the camera stream and restart vision.");
	}

	{
		std::lock_guard<std::mutex> lock(frameMutex);
		captureFinished = true;
	}
	frameReady.notify_all();
}

std::shared_ptr<RawImage> OpenCVDriver::readLatestImage() {
	cv::Mat frame;
	{
		std::unique_lock<std::mutex> lock(frameMutex);
		frameReady.wait(lock, [this] { return captureFinished || !latestFrame.empty(); });
		if(latestFrame.empty())
			return nullptr;
		std::swap(frame, latestFrame);
	}

	if(frame.type() != CV_8UC3) {
		WARN("OpenCV camera returned an unsupported pixel format; expected BGR8.");
		return nullptr;
	}
	if(image == nullptr || !image.unique() || image->width != frame.cols || image->height != frame.rows)
		image = std::make_shared<RawImage>(&PixelFormat::BGR8, frame.cols, frame.rows, name);

	// Keep OpenCL work on the processing thread, outside the capture lock,
	// so a GPU transfer cannot stop the camera reader draining the stream.
	CLMap<uint8_t> map = image->write<uint8_t>();
	cv::Mat mat(frame.size(), CV_8UC3, (void*)*map);
	frame.copyTo(mat);
	return image;
}

std::shared_ptr<RawImage> OpenCVDriver::readImage() {
	if(liveSource)
		return readLatestImage();

	// Recorded inputs remain synchronous so no frames are skipped.
	if(image == nullptr || !image.unique())
		image = std::make_shared<RawImage>(&PixelFormat::BGR8, capture.get(cv::CAP_PROP_FRAME_WIDTH), capture.get(cv::CAP_PROP_FRAME_HEIGHT), name);

	CLMap<uint8_t> map = image->write<uint8_t>();
	cv::Mat mat(cv::Size(image->width, image->height), CV_8UC3, (void*)*map);
	if(!capture.read(mat))
		return nullptr;

	return image;
}

const PixelFormat OpenCVDriver::format() {
	return PixelFormat::BGR8;
}

double OpenCVDriver::expectedFrametime() {
	return frameTime;
}


double OpenCVDriver::getTime() {
	// HTTP/RTSP streams and live devices must stay on the host clock even if
	// the decoder reports plausible frame counts after a settings change.
	if(liveSource)
		return getRealTime();

	const double pos = capture.get(cv::CAP_PROP_POS_FRAMES);
	const double fps = capture.get(cv::CAP_PROP_FPS);
	const double frameCount = capture.get(cv::CAP_PROP_FRAME_COUNT);

	// Live devices and network streams often report a frame position of zero
	// instead of -1. They may also expose an invalid FPS or frame count. Such
	// values are not timestamps and previously produced capture times near the
	// Unix epoch, which prevents multi-camera trackers from aligning frames.
	if(!std::isfinite(pos) || pos < 0.0 ||
			!std::isfinite(fps) || fps <= 0.0 ||
			!std::isfinite(frameCount) || frameCount <= 0.0)
		return getRealTime();

	return pos / fps;
}
