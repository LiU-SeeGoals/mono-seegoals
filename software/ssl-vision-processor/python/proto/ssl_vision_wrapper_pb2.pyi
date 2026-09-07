from proto import ssl_vision_detection_pb2 as _ssl_vision_detection_pb2
from proto import ssl_vision_geometry_pb2 as _ssl_vision_geometry_pb2
from proto import ssl_vp_config_pb2 as _ssl_vp_config_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor
SSL_SOURCE_ERFORCE_SIM: SSL_Source
SSL_SOURCE_GRSIM: SSL_Source
SSL_SOURCE_OTHER: SSL_Source
SSL_SOURCE_SSL_VISION: SSL_Source
SSL_SOURCE_UNKNOWN: SSL_Source
SSL_SOURCE_VISION_PROCESSOR: SSL_Source

class SSL_WrapperPacket(_message.Message):
    __slots__ = ["config", "detection", "geometry", "source"]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    DETECTION_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    config: _ssl_vp_config_pb2.SSL_VPConfig
    detection: _ssl_vision_detection_pb2.SSL_DetectionFrame
    geometry: _ssl_vision_geometry_pb2.SSL_GeometryData
    source: SSL_Source
    def __init__(self, detection: _Optional[_Union[_ssl_vision_detection_pb2.SSL_DetectionFrame, _Mapping]] = ..., geometry: _Optional[_Union[_ssl_vision_geometry_pb2.SSL_GeometryData, _Mapping]] = ..., source: _Optional[_Union[SSL_Source, str]] = ..., config: _Optional[_Union[_ssl_vp_config_pb2.SSL_VPConfig, _Mapping]] = ...) -> None: ...

class SSL_Source(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = []
