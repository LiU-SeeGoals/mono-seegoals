from proto import ssl_gc_common_pb2 as _ssl_gc_common_pb2
from proto import ssl_gc_geometry_pb2 as _ssl_gc_geometry_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

CAPABILITY_DETECT_FLYING_BALLS: Capability
CAPABILITY_DETECT_KICKED_BALLS: Capability
CAPABILITY_DETECT_MULTIPLE_BALLS: Capability
CAPABILITY_UNKNOWN: Capability
DESCRIPTOR: _descriptor.FileDescriptor

class KickedBall(_message.Message):
    __slots__ = ["pos", "robot_id", "start_timestamp", "stop_pos", "stop_timestamp", "vel"]
    POS_FIELD_NUMBER: _ClassVar[int]
    ROBOT_ID_FIELD_NUMBER: _ClassVar[int]
    START_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    STOP_POS_FIELD_NUMBER: _ClassVar[int]
    STOP_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    VEL_FIELD_NUMBER: _ClassVar[int]
    pos: _ssl_gc_geometry_pb2.Vector2
    robot_id: _ssl_gc_common_pb2.RobotId
    start_timestamp: float
    stop_pos: _ssl_gc_geometry_pb2.Vector2
    stop_timestamp: float
    vel: _ssl_gc_geometry_pb2.Vector3
    def __init__(self, pos: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., vel: _Optional[_Union[_ssl_gc_geometry_pb2.Vector3, _Mapping]] = ..., start_timestamp: _Optional[float] = ..., stop_timestamp: _Optional[float] = ..., stop_pos: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., robot_id: _Optional[_Union[_ssl_gc_common_pb2.RobotId, _Mapping]] = ...) -> None: ...

class TrackedBall(_message.Message):
    __slots__ = ["pos", "vel", "visibility"]
    POS_FIELD_NUMBER: _ClassVar[int]
    VEL_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    pos: _ssl_gc_geometry_pb2.Vector3
    vel: _ssl_gc_geometry_pb2.Vector3
    visibility: float
    def __init__(self, pos: _Optional[_Union[_ssl_gc_geometry_pb2.Vector3, _Mapping]] = ..., vel: _Optional[_Union[_ssl_gc_geometry_pb2.Vector3, _Mapping]] = ..., visibility: _Optional[float] = ...) -> None: ...

class TrackedFrame(_message.Message):
    __slots__ = ["balls", "capabilities", "frame_number", "kicked_ball", "robots", "timestamp"]
    BALLS_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    FRAME_NUMBER_FIELD_NUMBER: _ClassVar[int]
    KICKED_BALL_FIELD_NUMBER: _ClassVar[int]
    ROBOTS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    balls: _containers.RepeatedCompositeFieldContainer[TrackedBall]
    capabilities: _containers.RepeatedScalarFieldContainer[Capability]
    frame_number: int
    kicked_ball: KickedBall
    robots: _containers.RepeatedCompositeFieldContainer[TrackedRobot]
    timestamp: float
    def __init__(self, frame_number: _Optional[int] = ..., timestamp: _Optional[float] = ..., balls: _Optional[_Iterable[_Union[TrackedBall, _Mapping]]] = ..., robots: _Optional[_Iterable[_Union[TrackedRobot, _Mapping]]] = ..., kicked_ball: _Optional[_Union[KickedBall, _Mapping]] = ..., capabilities: _Optional[_Iterable[_Union[Capability, str]]] = ...) -> None: ...

class TrackedRobot(_message.Message):
    __slots__ = ["orientation", "pos", "robot_id", "vel", "vel_angular", "visibility"]
    ORIENTATION_FIELD_NUMBER: _ClassVar[int]
    POS_FIELD_NUMBER: _ClassVar[int]
    ROBOT_ID_FIELD_NUMBER: _ClassVar[int]
    VEL_ANGULAR_FIELD_NUMBER: _ClassVar[int]
    VEL_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    orientation: float
    pos: _ssl_gc_geometry_pb2.Vector2
    robot_id: _ssl_gc_common_pb2.RobotId
    vel: _ssl_gc_geometry_pb2.Vector2
    vel_angular: float
    visibility: float
    def __init__(self, robot_id: _Optional[_Union[_ssl_gc_common_pb2.RobotId, _Mapping]] = ..., pos: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., orientation: _Optional[float] = ..., vel: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., vel_angular: _Optional[float] = ..., visibility: _Optional[float] = ...) -> None: ...

class Capability(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = []
