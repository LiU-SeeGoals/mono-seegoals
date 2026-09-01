# SeeGoals ssl-vision-processor configuration

These files configure the replacement
[`ssl-vision-processor`](https://github.com/RoboCup-SSL/ssl-vision-processor)
for the two-camera SeeGoals field layout.

The calibration values were migrated from `robocup-ssl.xml`:

- camera 0 covers the negative-x (left) half of the field;
- camera 1 covers the positive-x (right) half of the field;
- both publish raw SSL detection packets to `224.5.23.2:10006`;
- the existing tracker/AutoRef remains responsible for tracked packets on
  `224.5.23.2:10010`.

## Camera authentication

The committed camera URLs intentionally contain no credentials. If the Axis
cameras require authentication, copy each camera file to an ignored local file
and put the credentials in the URL:

```bash
cp configs/vision-processor/camera-0.yml configs/vision-processor/camera-0.local.yml
cp configs/vision-processor/camera-1.yml configs/vision-processor/camera-1.local.yml
```

```text
http://USER:PASSWORD@192.168.1.10/mjpg/video.mjpg
```

Local overrides matching `camera-*.local.yml` are ignored by Git. Pass the
local filenames to `vision_processor` instead of the committed filenames.

## Start manually

Run these commands from the root of an `ssl-vision-processor` checkout after
copying the `configs/vision-processor` directory into it:

```bash
python3 python/geom_publisher.py configs/vision-processor/geometry.yml
```

In two further terminals:

```bash
./build/vision_processor configs/vision-processor/camera-0.yml
./build/vision_processor configs/vision-processor/camera-1.yml
```

Inspect both debug streams with:

```bash
python3 python/cam_viewer.py --cameras 2
```

The legacy launcher starts three cameras, but its saved field geometry declares
two. The new processor's automatic layout supports the two half-field cameras;
the overlapping centre camera must not be enabled without adding explicit
per-camera field extents upstream.
