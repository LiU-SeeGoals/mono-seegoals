# SeeGoals ssl-vision-processor configuration

These files configure the replacement
[`ssl-vision-processor`](https://github.com/RoboCup-SSL/ssl-vision-processor)
for the three-camera SeeGoals field layout.

The calibration values were migrated from `robocup-ssl.xml`:

- camera 0 covers the negative-x (left) half of the field;
- camera 1 covers the positive-x (right) half of the field;
- camera 2 overlaps both cameras across the field centre;
- all three publish raw SSL detection packets to `224.5.23.2:10006`;
- the existing tracker/AutoRef remains responsible for tracked packets on
  `224.5.23.2:10010`.

The complete camera models in `geometry.yml` were also migrated from the
legacy configuration. They must remain present: the upstream automatic camera
grid cannot represent the overlapping centre camera and otherwise produces a
very narrow, tall reprojected image.

## Camera authentication

The committed camera URLs intentionally contain no credentials. If the Axis
cameras require authentication, copy each camera file to an ignored local file
and put the credentials in the URL:

```bash
cp configs/vision-processor/camera-0.yml configs/vision-processor/camera-0.local.yml
cp configs/vision-processor/camera-1.yml configs/vision-processor/camera-1.local.yml
cp configs/vision-processor/camera-2.yml configs/vision-processor/camera-2.local.yml
```

```text
http://USER:PASSWORD@192.168.1.10/mjpg/video.mjpg
```

Camera 2 uses the Axis CGI endpoint:

```text
http://USER:PASSWORD@192.168.1.12/axis-cgi/mjpg/video.cgi
```

Local overrides matching `camera-*.local.yml` are ignored by Git. Pass the
local filenames to `vision_processor` instead of the committed filenames.

## Start manually

Run these commands from the root of an `ssl-vision-processor` checkout after
copying the `configs/vision-processor` directory into it. Start the viewer
first so it receives the initial H.264 parameters:

```bash
python3 python/cam_viewer.py --cameras 3
```

Then start the geometry publisher. It must use this `geometry.yml` so the saved
three-camera models are published:

```bash
python3 python/geom_publisher.py configs/vision-processor/geometry.yml
```

In three further terminals:

```bash
./build/vision_processor configs/vision-processor/camera-0.local.yml
./build/vision_processor configs/vision-processor/camera-1.local.yml
./build/vision_processor configs/vision-processor/camera-2.local.yml
```

The committed configs use `stream.raw_feed: false`, so the debug streams cycle
through the raw, reprojected, gradient, and blob-score views. Temporarily set
it to `true` in a local file only when testing the camera input.
