from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

BLUE: Team
DESCRIPTOR: _descriptor.FileDescriptor
DIV_A: Division
DIV_B: Division
DIV_UNKNOWN: Division
UNKNOWN: Team
YELLOW: Team

class RobotId(_message.Message):
    __slots__ = ["id", "team"]
    ID_FIELD_NUMBER: _ClassVar[int]
    TEAM_FIELD_NUMBER: _ClassVar[int]
    id: int
    team: Team
    def __init__(self, id: _Optional[int] = ..., team: _Optional[_Union[Team, str]] = ...) -> None: ...

class Team(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = []

class Division(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = []
