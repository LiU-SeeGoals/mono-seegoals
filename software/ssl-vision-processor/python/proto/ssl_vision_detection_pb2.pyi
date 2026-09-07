from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SSL_DetectionBall(_message.Message):
    __slots__ = ["area", "confidence", "pixel_x", "pixel_y", "x", "y", "z"]
    AREA_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    PIXEL_X_FIELD_NUMBER: _ClassVar[int]
    PIXEL_Y_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    Z_FIELD_NUMBER: _ClassVar[int]
    area: int
    confidence: float
    pixel_x: float
    pixel_y: float
    x: float
    y: float
    z: float
    def __init__(self, confidence: _Optional[float] = ..., area: _Optional[int] = ..., x: _Optional[float] = ..., y: _Optional[float] = ..., z: _Optional[float] = ..., pixel_x: _Optional[float] = ..., pixel_y: _Optional[float] = ...) -> None: ...

class SSL_DetectionFrame(_message.Message):
    __slots__ = ["balls", "camera_id", "frame_number", "robots_blue", "robots_yellow", "t_capture", "t_capture_camera", "t_offsets", "t_sent"]
    BALLS_FIELD_NUMBER: _ClassVar[int]
    CAMERA_ID_FIELD_NUMBER: _ClassVar[int]
    FRAME_NUMBER_FIELD_NUMBER: _ClassVar[int]
    ROBOTS_BLUE_FIELD_NUMBER: _ClassVar[int]
    ROBOTS_YELLOW_FIELD_NUMBER: _ClassVar[int]
    T_CAPTURE_CAMERA_FIELD_NUMBER: _ClassVar[int]
    T_CAPTURE_FIELD_NUMBER: _ClassVar[int]
    T_OFFSETS_FIELD_NUMBER: _ClassVar[int]
    T_SENT_FIELD_NUMBER: _ClassVar[int]
    balls: _containers.RepeatedCompositeFieldContainer[SSL_DetectionBall]
    camera_id: int
    frame_number: int
    robots_blue: _containers.RepeatedCompositeFieldContainer[SSL_DetectionRobot]
    robots_yellow: _containers.RepeatedCompositeFieldContainer[SSL_DetectionRobot]
    t_capture: float
    t_capture_camera: float
    t_offsets: _containers.RepeatedScalarFieldContainer[float]
    t_sent: float
    def __init__(self, frame_number: _Optional[int] = ..., t_capture: _Optional[float] = ..., t_sent: _Optional[float] = ..., t_capture_camera: _Optional[float] = ..., t_offsets: _Optional[_Iterable[float]] = ..., camera_id: _Optional[int] = ..., balls: _Optional[_Iterable[_Union[SSL_DetectionBall, _Mapping]]] = ..., robots_yellow: _Optional[_Iterable[_Union[SSL_DetectionRobot, _Mapping]]] = ..., robots_blue: _Optional[_Iterable[_Union[SSL_DetectionRobot, _Mapping]]] = ...) -> None: ...

class SSL_DetectionRobot(_message.Message):
    __slots__ = ["confidence", "height", "orientation", "pixel_x", "pixel_y", "robot_id", "x", "y"]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    ORIENTATION_FIELD_NUMBER: _ClassVar[int]
    PIXEL_X_FIELD_NUMBER: _ClassVar[int]
    PIXEL_Y_FIELD_NUMBER: _ClassVar[int]
    ROBOT_ID_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    confidence: float
    height: float
    orientation: float
    pixel_x: float
    pixel_y: float
    robot_id: int
    x: float
    y: float
    def __init__(self, confidence: _Optional[float] = ..., robot_id: _Optional[int] = ..., x: _Optional[float] = ..., y: _Optional[float] = ..., orientation: _Optional[float] = ..., pixel_x: _Optional[float] = ..., pixel_y: _Optional[float] = ..., height: _Optional[float] = ...) -> None: ...
