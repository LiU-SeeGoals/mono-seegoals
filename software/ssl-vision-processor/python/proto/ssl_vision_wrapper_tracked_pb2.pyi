from proto import ssl_vision_detection_tracked_pb2 as _ssl_vision_detection_tracked_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TrackerWrapperPacket(_message.Message):
    __slots__ = ["source_name", "tracked_frame", "uuid"]
    SOURCE_NAME_FIELD_NUMBER: _ClassVar[int]
    TRACKED_FRAME_FIELD_NUMBER: _ClassVar[int]
    UUID_FIELD_NUMBER: _ClassVar[int]
    source_name: str
    tracked_frame: _ssl_vision_detection_tracked_pb2.TrackedFrame
    uuid: str
    def __init__(self, uuid: _Optional[str] = ..., source_name: _Optional[str] = ..., tracked_frame: _Optional[_Union[_ssl_vision_detection_tracked_pb2.TrackedFrame, _Mapping]] = ...) -> None: ...
