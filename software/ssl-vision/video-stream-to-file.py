#!/usr/bin/env python3

import cv2
import requests
import numpy as np
import base64
import sys
import os
import time
import signal
from pathlib import Path

class MJPEGCameraCapture:
    def __init__(self, camera_number, base_url="192.168.1", fps_limit=30):
        self.camera_number = camera_number
        self.url = f"http://{base_url}.{camera_number}/mjpg/video.mjpg"
        self.script_dir = Path(__file__).parent.resolve()
        self.output_file = self.script_dir / f"camera{camera_number}.base64"
        self.fps_limit = fps_limit
        self.frame_delay = 1.0 / fps_limit if fps_limit > 0 else 0
        self.running = True
        self.session = None
        
        # Get credentials from environment or use defaults (not recommended)
        self.username = os.getenv('CAMERA_USER', 'root')
        self.password = os.getenv('CAMERA_PASS', 'raswa151')
        
        # Setup signal handlers for graceful shutdown
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)
    
    def _signal_handler(self, signum, frame):
        """Handle shutdown signals gracefully"""
        print(f"\nReceived signal {signum}, shutting down...")
        self.running = False
    
    def _connect(self, timeout=10, max_retries=3):
        """Establish connection to camera with retry logic"""
        for attempt in range(max_retries):
            try:
                if self.session:
                    self.session.close()
                
                self.session = requests.Session()
                response = self.session.get(
                    self.url,
                    auth=(self.username, self.password),
                    stream=True,
                    timeout=timeout
                )
                
                if response.status_code == 200:
                    print(f"Connected to camera {self.camera_number}")
                    return response
                else:
                    print(f"HTTP {response.status_code} from camera {self.camera_number}")
                    
            except requests.exceptions.RequestException as e:
                print(f"Connection attempt {attempt + 1}/{max_retries} failed: {e}")
                if attempt < max_retries - 1:
                    time.sleep(2 ** attempt)  # Exponential backoff
        
        raise ConnectionError(f"Failed to connect to camera {self.camera_number} after {max_retries} attempts")
    
    def _extract_jpeg_from_buffer(self, buffer):
        """Extract JPEG image from buffer"""
        start = buffer.find(b'\xff\xd8')
        end = buffer.find(b'\xff\xd9')
        
        if start != -1 and end != -1:
            jpeg_data = buffer[start:end + 2]
            remaining_buffer = buffer[end + 2:]
            return jpeg_data, remaining_buffer
        
        return None, buffer
    
    def _save_frame(self, jpeg_data):
        """Decode, convert, and save frame as base64"""
        try:
            img = cv2.imdecode(np.frombuffer(jpeg_data, dtype=np.uint8), cv2.IMREAD_COLOR)
            
            if img is None:
                print("Failed to decode JPEG data")
                return False
            
            # Convert BGR to RGB
            img_rgb = cv2.cvtColor(img, cv2.COLOR_BGR2RGB)
            
            # Re-encode as JPEG
            success, encoded_jpg = cv2.imencode('.jpg', img_rgb)
            
            if not success:
                print("Failed to encode image")
                return False
            
            # Convert to base64
            encoded_img = base64.b64encode(encoded_jpg).decode('utf-8')
            
            # Write to file atomically
            temp_file = self.output_file.with_suffix('.tmp')
            temp_file.write_text(encoded_img)
            temp_file.rename(self.output_file)
            
            return True
            
        except Exception as e:
            print(f"Error saving frame: {e}")
            return False
    
    def run(self):
        """Main capture loop with automatic reconnection"""
        frame_buffer = b''
        last_frame_time = 0
        consecutive_errors = 0
        max_consecutive_errors = 10
        max_buffer_size = 10 * 1024 * 1024  # 10MB buffer limit
        
        while self.running:
            try:
                # Connect or reconnect to camera
                response = self._connect()
                consecutive_errors = 0
                frame_buffer = b''
                
                # Read from stream
                for chunk in response.iter_content(chunk_size=4096):
                    if not self.running:
                        break
                    
                    if not chunk:
                        print("Empty chunk received, reconnecting...")
                        break
                    
                    frame_buffer += chunk
                    
                    # Prevent buffer from growing unbounded
                    if len(frame_buffer) > max_buffer_size:
                        print("Buffer overflow, resetting...")
                        frame_buffer = b''
                        continue
                    
                    # Try to extract a complete JPEG
                    jpeg_data, frame_buffer = self._extract_jpeg_from_buffer(frame_buffer)
                    
                    if jpeg_data:
                        # Respect FPS limit
                        current_time = time.time()
                        elapsed = current_time - last_frame_time
                        
                        if elapsed < self.frame_delay:
                            time.sleep(self.frame_delay - elapsed)
                        
                        # Save the frame
                        if self._save_frame(jpeg_data):
                            last_frame_time = time.time()
                            consecutive_errors = 0
                        else:
                            consecutive_errors += 1
                            
                        if consecutive_errors >= max_consecutive_errors:
                            print("Too many consecutive decode errors, reconnecting...")
                            break
                
            except ConnectionError as e:
                print(f"Connection error: {e}")
                if self.running:
                    print("Retrying in 5 seconds...")
                    time.sleep(5)
                    
            except Exception as e:
                print(f"Unexpected error: {e}")
                if self.running:
                    print("Retrying in 5 seconds...")
                    time.sleep(5)
            
            finally:
                if self.session:
                    try:
                        self.session.close()
                    except:
                        pass
        
        print("Camera capture stopped")


def main():
    if len(sys.argv) < 2:
        print("Usage: ./camera_capture.py <camera_number> [fps_limit]")
        print("Example: ./camera_capture.py 100 30")
        print("\nEnvironment variables:")
        print("  CAMERA_USER - Camera username (default: root)")
        print("  CAMERA_PASS - Camera password (default: raswa151)")
        sys.exit(1)
    
    camera_number = sys.argv[1]
    fps_limit = int(sys.argv[2]) if len(sys.argv) > 2 else 30
    
    try:
        capture = MJPEGCameraCapture(camera_number, fps_limit=fps_limit)
        capture.run()
    except KeyboardInterrupt:
        print("\nShutdown requested by user")
    except Exception as e:
        print(f"Fatal error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()