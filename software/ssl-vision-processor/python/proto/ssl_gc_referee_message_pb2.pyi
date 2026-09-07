from proto import ssl_gc_game_event_pb2 as _ssl_gc_game_event_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor
ELIMINATION_PHASE: MatchType
FRIENDLY: MatchType
GROUP_PHASE: MatchType
UNKNOWN_MATCH: MatchType

class GameEventProposalGroup(_message.Message):
    __slots__ = ["accepted", "game_event"]
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    GAME_EVENT_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    game_event: _containers.RepeatedCompositeFieldContainer[_ssl_gc_game_event_pb2.GameEvent]
    def __init__(self, game_event: _Optional[_Iterable[_Union[_ssl_gc_game_event_pb2.GameEvent, _Mapping]]] = ..., accepted: bool = ...) -> None: ...

class Referee(_message.Message):
    __slots__ = ["blue", "blue_team_on_positive_half", "command", "command_counter", "command_timestamp", "current_action_time_remaining", "designated_position", "game_event_proposals", "game_events", "match_type", "next_command", "packet_timestamp", "source_identifier", "stage", "stage_time_left", "yellow"]
    class Command(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = []
    class Stage(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = []
    class Point(_message.Message):
        __slots__ = ["x", "y"]
        X_FIELD_NUMBER: _ClassVar[int]
        Y_FIELD_NUMBER: _ClassVar[int]
        x: float
        y: float
        def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ...) -> None: ...
    class TeamInfo(_message.Message):
        __slots__ = ["ball_placement_failures", "ball_placement_failures_reached", "bot_substitution_allowed", "bot_substitution_intent", "can_place_ball", "foul_counter", "goalkeeper", "max_allowed_bots", "name", "red_cards", "score", "timeout_time", "timeouts", "yellow_card_times", "yellow_cards"]
        BALL_PLACEMENT_FAILURES_FIELD_NUMBER: _ClassVar[int]
        BALL_PLACEMENT_FAILURES_REACHED_FIELD_NUMBER: _ClassVar[int]
        BOT_SUBSTITUTION_ALLOWED_FIELD_NUMBER: _ClassVar[int]
        BOT_SUBSTITUTION_INTENT_FIELD_NUMBER: _ClassVar[int]
        CAN_PLACE_BALL_FIELD_NUMBER: _ClassVar[int]
        FOUL_COUNTER_FIELD_NUMBER: _ClassVar[int]
        GOALKEEPER_FIELD_NUMBER: _ClassVar[int]
        MAX_ALLOWED_BOTS_FIELD_NUMBER: _ClassVar[int]
        NAME_FIELD_NUMBER: _ClassVar[int]
        RED_CARDS_FIELD_NUMBER: _ClassVar[int]
        SCORE_FIELD_NUMBER: _ClassVar[int]
        TIMEOUTS_FIELD_NUMBER: _ClassVar[int]
        TIMEOUT_TIME_FIELD_NUMBER: _ClassVar[int]
        YELLOW_CARDS_FIELD_NUMBER: _ClassVar[int]
        YELLOW_CARD_TIMES_FIELD_NUMBER: _ClassVar[int]
        ball_placement_failures: int
        ball_placement_failures_reached: bool
        bot_substitution_allowed: bool
        bot_substitution_intent: bool
        can_place_ball: bool
        foul_counter: int
        goalkeeper: int
        max_allowed_bots: int
        name: str
        red_cards: int
        score: int
        timeout_time: int
        timeouts: int
        yellow_card_times: _containers.RepeatedScalarFieldContainer[int]
        yellow_cards: int
        def __init__(self, name: _Optional[str] = ..., score: _Optional[int] = ..., red_cards: _Optional[int] = ..., yellow_card_times: _Optional[_Iterable[int]] = ..., yellow_cards: _Optional[int] = ..., timeouts: _Optional[int] = ..., timeout_time: _Optional[int] = ..., goalkeeper: _Optional[int] = ..., foul_counter: _Optional[int] = ..., ball_placement_failures: _Optional[int] = ..., can_place_ball: bool = ..., max_allowed_bots: _Optional[int] = ..., bot_substitution_intent: bool = ..., ball_placement_failures_reached: bool = ..., bot_substitution_allowed: bool = ...) -> None: ...
    BALL_PLACEMENT_BLUE: Referee.Command
    BALL_PLACEMENT_YELLOW: Referee.Command
    BLUE_FIELD_NUMBER: _ClassVar[int]
    BLUE_TEAM_ON_POSITIVE_HALF_FIELD_NUMBER: _ClassVar[int]
    COMMAND_COUNTER_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    COMMAND_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    CURRENT_ACTION_TIME_REMAINING_FIELD_NUMBER: _ClassVar[int]
    DESIGNATED_POSITION_FIELD_NUMBER: _ClassVar[int]
    DIRECT_FREE_BLUE: Referee.Command
    DIRECT_FREE_YELLOW: Referee.Command
    EXTRA_FIRST_HALF: Referee.Stage
    EXTRA_FIRST_HALF_PRE: Referee.Stage
    EXTRA_HALF_TIME: Referee.Stage
    EXTRA_SECOND_HALF: Referee.Stage
    EXTRA_SECOND_HALF_PRE: Referee.Stage
    EXTRA_TIME_BREAK: Referee.Stage
    FORCE_START: Referee.Command
    GAME_EVENTS_FIELD_NUMBER: _ClassVar[int]
    GAME_EVENT_PROPOSALS_FIELD_NUMBER: _ClassVar[int]
    HALT: Referee.Command
    MATCH_TYPE_FIELD_NUMBER: _ClassVar[int]
    NEXT_COMMAND_FIELD_NUMBER: _ClassVar[int]
    NORMAL_FIRST_HALF: Referee.Stage
    NORMAL_FIRST_HALF_PRE: Referee.Stage
    NORMAL_HALF_TIME: Referee.Stage
    NORMAL_SECOND_HALF: Referee.Stage
    NORMAL_SECOND_HALF_PRE: Referee.Stage
    NORMAL_START: Referee.Command
    PACKET_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    PENALTY_SHOOTOUT: Referee.Stage
    PENALTY_SHOOTOUT_BREAK: Referee.Stage
    POST_GAME: Referee.Stage
    PREPARE_KICKOFF_BLUE: Referee.Command
    PREPARE_KICKOFF_YELLOW: Referee.Command
    PREPARE_PENALTY_BLUE: Referee.Command
    PREPARE_PENALTY_YELLOW: Referee.Command
    SOURCE_IDENTIFIER_FIELD_NUMBER: _ClassVar[int]
    STAGE_FIELD_NUMBER: _ClassVar[int]
    STAGE_TIME_LEFT_FIELD_NUMBER: _ClassVar[int]
    STOP: Referee.Command
    TIMEOUT_BLUE: Referee.Command
    TIMEOUT_YELLOW: Referee.Command
    YELLOW_FIELD_NUMBER: _ClassVar[int]
    blue: Referee.TeamInfo
    blue_team_on_positive_half: bool
    command: Referee.Command
    command_counter: int
    command_timestamp: int
    current_action_time_remaining: int
    designated_position: Referee.Point
    game_event_proposals: _containers.RepeatedCompositeFieldContainer[GameEventProposalGroup]
    game_events: _containers.RepeatedCompositeFieldContainer[_ssl_gc_game_event_pb2.GameEvent]
    match_type: MatchType
    next_command: Referee.Command
    packet_timestamp: int
    source_identifier: str
    stage: Referee.Stage
    stage_time_left: int
    yellow: Referee.TeamInfo
    def __init__(self, source_identifier: _Optional[str] = ..., match_type: _Optional[_Union[MatchType, str]] = ..., packet_timestamp: _Optional[int] = ..., stage: _Optional[_Union[Referee.Stage, str]] = ..., stage_time_left: _Optional[int] = ..., command: _Optional[_Union[Referee.Command, str]] = ..., command_counter: _Optional[int] = ..., command_timestamp: _Optional[int] = ..., yellow: _Optional[_Union[Referee.TeamInfo, _Mapping]] = ..., blue: _Optional[_Union[Referee.TeamInfo, _Mapping]] = ..., designated_position: _Optional[_Union[Referee.Point, _Mapping]] = ..., blue_team_on_positive_half: bool = ..., next_command: _Optional[_Union[Referee.Command, str]] = ..., game_events: _Optional[_Iterable[_Union[_ssl_gc_game_event_pb2.GameEvent, _Mapping]]] = ..., game_event_proposals: _Optional[_Iterable[_Union[GameEventProposalGroup, _Mapping]]] = ..., current_action_time_remaining: _Optional[int] = ...) -> None: ...

class MatchType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = []
