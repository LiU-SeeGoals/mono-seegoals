from proto import ssl_gc_geometry_pb2 as _ssl_gc_geometry_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor
INDOOR: SSL_VpConfigCameraWBType
MANUAL: SSL_VpConfigCameraWBType
MVIMPACT: SSL_VPConfigCameraDriver
OPENCV: SSL_VPConfigCameraDriver
OUTDOOR: SSL_VpConfigCameraWBType
RAW: SSL_VPConfigStreamType
REPROJ_BLOB_SCORE: SSL_VPConfigStreamType
REPROJ_FALSE_COLOR: SSL_VPConfigStreamType
REPROJ_GRADIENT_DOT: SSL_VPConfigStreamType
SPINNAKER: SSL_VPConfigCameraDriver

class Color(_message.Message):
    __slots__ = ["blue", "green", "red"]
    BLUE_FIELD_NUMBER: _ClassVar[int]
    GREEN_FIELD_NUMBER: _ClassVar[int]
    RED_FIELD_NUMBER: _ClassVar[int]
    blue: int
    green: int
    red: int
    def __init__(self, red: _Optional[int] = ..., green: _Optional[int] = ..., blue: _Optional[int] = ...) -> None: ...

class SSL_VPConfig(_message.Message):
    __slots__ = ["camera", "camera_id", "color", "geometry", "instance", "network", "robot_heights", "stream", "thresholds", "wait_for_geometry"]
    class RobotHeightsEntry(_message.Message):
        __slots__ = ["key", "value"]
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    CAMERA_FIELD_NUMBER: _ClassVar[int]
    CAMERA_ID_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    ROBOT_HEIGHTS_FIELD_NUMBER: _ClassVar[int]
    STREAM_FIELD_NUMBER: _ClassVar[int]
    THRESHOLDS_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_GEOMETRY_FIELD_NUMBER: _ClassVar[int]
    camera: SSL_VPConfigCamera
    camera_id: int
    color: SSL_VPConfigColor
    geometry: SSL_VPConfigGeometry
    instance: str
    network: SSL_VPConfigNetwork
    robot_heights: _containers.ScalarMap[str, float]
    stream: SSL_VPConfigStream
    thresholds: SSL_VPConfigThresholds
    wait_for_geometry: bool
    def __init__(self, instance: _Optional[str] = ..., camera_id: _Optional[int] = ..., robot_heights: _Optional[_Mapping[str, float]] = ..., camera: _Optional[_Union[SSL_VPConfigCamera, _Mapping]] = ..., geometry: _Optional[_Union[SSL_VPConfigGeometry, _Mapping]] = ..., thresholds: _Optional[_Union[SSL_VPConfigThresholds, _Mapping]] = ..., color: _Optional[_Union[SSL_VPConfigColor, _Mapping]] = ..., network: _Optional[_Union[SSL_VPConfigNetwork, _Mapping]] = ..., stream: _Optional[_Union[SSL_VPConfigStream, _Mapping]] = ..., wait_for_geometry: bool = ...) -> None: ...

class SSL_VPConfigCamera(_message.Message):
    __slots__ = ["driver", "exposure", "gain", "gamma", "height", "id", "path", "wb_blue", "wb_red", "wb_type", "width"]
    DRIVER_FIELD_NUMBER: _ClassVar[int]
    EXPOSURE_FIELD_NUMBER: _ClassVar[int]
    GAIN_FIELD_NUMBER: _ClassVar[int]
    GAMMA_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    WB_BLUE_FIELD_NUMBER: _ClassVar[int]
    WB_RED_FIELD_NUMBER: _ClassVar[int]
    WB_TYPE_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    driver: SSL_VPConfigCameraDriver
    exposure: float
    gain: float
    gamma: float
    height: int
    id: int
    path: str
    wb_blue: float
    wb_red: float
    wb_type: SSL_VpConfigCameraWBType
    width: int
    def __init__(self, driver: _Optional[_Union[SSL_VPConfigCameraDriver, str]] = ..., id: _Optional[int] = ..., path: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., exposure: _Optional[float] = ..., gain: _Optional[float] = ..., gamma: _Optional[float] = ..., wb_type: _Optional[_Union[SSL_VpConfigCameraWBType, str]] = ..., wb_red: _Optional[float] = ..., wb_blue: _Optional[float] = ...) -> None: ...

class SSL_VPConfigColor(_message.Message):
    __slots__ = ["blue", "field", "green", "history_force", "orange", "pink", "reference_force", "yellow"]
    BLUE_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    GREEN_FIELD_NUMBER: _ClassVar[int]
    HISTORY_FORCE_FIELD_NUMBER: _ClassVar[int]
    ORANGE_FIELD_NUMBER: _ClassVar[int]
    PINK_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FORCE_FIELD_NUMBER: _ClassVar[int]
    YELLOW_FIELD_NUMBER: _ClassVar[int]
    blue: VPColor
    field: VPColor
    green: VPColor
    history_force: float
    orange: VPColor
    pink: VPColor
    reference_force: float
    yellow: VPColor
    def __init__(self, reference_force: _Optional[float] = ..., history_force: _Optional[float] = ..., orange: _Optional[_Union[VPColor, _Mapping]] = ..., field: _Optional[_Union[VPColor, _Mapping]] = ..., yellow: _Optional[_Union[VPColor, _Mapping]] = ..., blue: _Optional[_Union[VPColor, _Mapping]] = ..., green: _Optional[_Union[VPColor, _Mapping]] = ..., pink: _Optional[_Union[VPColor, _Mapping]] = ...) -> None: ...

class SSL_VPConfigGeometry(_message.Message):
    __slots__ = ["camera_amount", "camera_height", "line_corners", "refinement"]
    CAMERA_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    CAMERA_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    LINE_CORNERS_FIELD_NUMBER: _ClassVar[int]
    REFINEMENT_FIELD_NUMBER: _ClassVar[int]
    camera_amount: int
    camera_height: float
    line_corners: _containers.RepeatedCompositeFieldContainer[_ssl_gc_geometry_pb2.Vector2]
    refinement: bool
    def __init__(self, camera_amount: _Optional[int] = ..., camera_height: _Optional[float] = ..., line_corners: _Optional[_Iterable[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]]] = ..., refinement: bool = ...) -> None: ...

class SSL_VPConfigNetwork(_message.Message):
    __slots__ = ["gc_ip", "gc_port", "vision_ip"]
    GC_IP_FIELD_NUMBER: _ClassVar[int]
    GC_PORT_FIELD_NUMBER: _ClassVar[int]
    VISION_IP_FIELD_NUMBER: _ClassVar[int]
    gc_ip: str
    gc_port: int
    vision_ip: str
    def __init__(self, gc_ip: _Optional[str] = ..., gc_port: _Optional[int] = ..., vision_ip: _Optional[str] = ...) -> None: ...

class SSL_VPConfigStream(_message.Message):
    __slots__ = ["active", "stream_ip", "stream_port", "type"]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    STREAM_IP_FIELD_NUMBER: _ClassVar[int]
    STREAM_PORT_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    active: bool
    stream_ip: str
    stream_port: int
    type: SSL_VPConfigStreamType
    def __init__(self, active: bool = ..., type: _Optional[_Union[SSL_VPConfigStreamType, str]] = ..., stream_ip: _Optional[str] = ..., stream_port: _Optional[int] = ...) -> None: ...

class SSL_VPConfigThresholds(_message.Message):
    __slots__ = ["circularity", "clipping_tolerance", "geometry_tolerance", "min_cam_edge_distance", "min_confidence", "score"]
    CIRCULARITY_FIELD_NUMBER: _ClassVar[int]
    CLIPPING_TOLERANCE_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_TOLERANCE_FIELD_NUMBER: _ClassVar[int]
    MIN_CAM_EDGE_DISTANCE_FIELD_NUMBER: _ClassVar[int]
    MIN_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    circularity: float
    clipping_tolerance: float
    geometry_tolerance: float
    min_cam_edge_distance: float
    min_confidence: float
    score: float
    def __init__(self, circularity: _Optional[float] = ..., score: _Optional[float] = ..., min_confidence: _Optional[float] = ..., min_cam_edge_distance: _Optional[float] = ..., clipping_tolerance: _Optional[float] = ..., geometry_tolerance: _Optional[float] = ...) -> None: ...

class VPColor(_message.Message):
    __slots__ = ["live", "reference"]
    LIVE_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    live: Color
    reference: Color
    def __init__(self, reference: _Optional[_Union[Color, _Mapping]] = ..., live: _Optional[_Union[Color, _Mapping]] = ...) -> None: ...

class SSL_VPConfigCameraDriver(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = []

class SSL_VpConfigCameraWBType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = []

class SSL_VPConfigStreamType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = []
