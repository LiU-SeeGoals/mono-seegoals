from proto import ssl_gc_common_pb2 as _ssl_gc_common_pb2
from proto import ssl_gc_geometry_pb2 as _ssl_gc_geometry_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GameEvent(_message.Message):
    __slots__ = ["aimless_kick", "attacker_double_touched_ball", "attacker_too_close_to_defense_area", "attacker_touched_ball_in_defense_area", "ball_left_field_goal_line", "ball_left_field_touch_line", "bot_crash_drawn", "bot_crash_unique", "bot_dribbled_ball_too_far", "bot_held_ball_deliberately", "bot_interfered_placement", "bot_kicked_ball_too_fast", "bot_pushed_bot", "bot_substitution", "bot_tipped_over", "bot_too_fast_in_stop", "boundary_crossing", "challenge_flag", "challenge_flag_handled", "created_timestamp", "defender_in_defense_area", "defender_too_close_to_kick_point", "emergency_stop", "goal", "invalid_goal", "keeper_held_ball", "multiple_cards", "multiple_fouls", "no_progress_in_game", "origin", "penalty_kick_failed", "placement_failed", "placement_succeeded", "possible_goal", "too_many_robots", "type", "unsporting_behavior_major", "unsporting_behavior_minor"]
    class Type(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = []
    class AimlessKick(_message.Message):
        __slots__ = ["by_bot", "by_team", "kick_location", "location"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        KICK_LOCATION_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        kick_location: _ssl_gc_geometry_pb2.Vector2
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., kick_location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ...) -> None: ...
    class AttackerDoubleTouchedBall(_message.Message):
        __slots__ = ["by_bot", "by_team", "location"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ...) -> None: ...
    class AttackerTooCloseToDefenseArea(_message.Message):
        __slots__ = ["ball_location", "by_bot", "by_team", "distance", "location"]
        BALL_LOCATION_FIELD_NUMBER: _ClassVar[int]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        DISTANCE_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        ball_location: _ssl_gc_geometry_pb2.Vector2
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        distance: float
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., distance: _Optional[float] = ..., ball_location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ...) -> None: ...
    class AttackerTouchedBallInDefenseArea(_message.Message):
        __slots__ = ["by_bot", "by_team", "distance", "location"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        DISTANCE_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        distance: float
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., distance: _Optional[float] = ...) -> None: ...
    class AttackerTouchedOpponentInDefenseArea(_message.Message):
        __slots__ = ["by_bot", "by_team", "location", "victim"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        VICTIM_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        location: _ssl_gc_geometry_pb2.Vector2
        victim: int
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., victim: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ...) -> None: ...
    class BallLeftField(_message.Message):
        __slots__ = ["by_bot", "by_team", "location"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ...) -> None: ...
    class BotCrashDrawn(_message.Message):
        __slots__ = ["bot_blue", "bot_yellow", "crash_angle", "crash_speed", "location", "speed_diff"]
        BOT_BLUE_FIELD_NUMBER: _ClassVar[int]
        BOT_YELLOW_FIELD_NUMBER: _ClassVar[int]
        CRASH_ANGLE_FIELD_NUMBER: _ClassVar[int]
        CRASH_SPEED_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        SPEED_DIFF_FIELD_NUMBER: _ClassVar[int]
        bot_blue: int
        bot_yellow: int
        crash_angle: float
        crash_speed: float
        location: _ssl_gc_geometry_pb2.Vector2
        speed_diff: float
        def __init__(self, bot_yellow: _Optional[int] = ..., bot_blue: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., crash_speed: _Optional[float] = ..., speed_diff: _Optional[float] = ..., crash_angle: _Optional[float] = ...) -> None: ...
    class BotCrashUnique(_message.Message):
        __slots__ = ["by_team", "crash_angle", "crash_speed", "location", "speed_diff", "victim", "violator"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        CRASH_ANGLE_FIELD_NUMBER: _ClassVar[int]
        CRASH_SPEED_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        SPEED_DIFF_FIELD_NUMBER: _ClassVar[int]
        VICTIM_FIELD_NUMBER: _ClassVar[int]
        VIOLATOR_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        crash_angle: float
        crash_speed: float
        location: _ssl_gc_geometry_pb2.Vector2
        speed_diff: float
        victim: int
        violator: int
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., violator: _Optional[int] = ..., victim: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., crash_speed: _Optional[float] = ..., speed_diff: _Optional[float] = ..., crash_angle: _Optional[float] = ...) -> None: ...
    class BotDribbledBallTooFar(_message.Message):
        __slots__ = ["by_bot", "by_team", "end", "start"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        END_FIELD_NUMBER: _ClassVar[int]
        START_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        end: _ssl_gc_geometry_pb2.Vector2
        start: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., start: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., end: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ...) -> None: ...
    class BotHeldBallDeliberately(_message.Message):
        __slots__ = ["by_bot", "by_team", "duration", "location"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        DURATION_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        duration: float
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., duration: _Optional[float] = ...) -> None: ...
    class BotInterferedPlacement(_message.Message):
        __slots__ = ["by_bot", "by_team", "location"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ...) -> None: ...
    class BotKickedBallTooFast(_message.Message):
        __slots__ = ["by_bot", "by_team", "chipped", "initial_ball_speed", "location"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        CHIPPED_FIELD_NUMBER: _ClassVar[int]
        INITIAL_BALL_SPEED_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        chipped: bool
        initial_ball_speed: float
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., initial_ball_speed: _Optional[float] = ..., chipped: bool = ...) -> None: ...
    class BotPushedBot(_message.Message):
        __slots__ = ["by_team", "location", "pushed_distance", "victim", "violator"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        PUSHED_DISTANCE_FIELD_NUMBER: _ClassVar[int]
        VICTIM_FIELD_NUMBER: _ClassVar[int]
        VIOLATOR_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        location: _ssl_gc_geometry_pb2.Vector2
        pushed_distance: float
        victim: int
        violator: int
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., violator: _Optional[int] = ..., victim: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., pushed_distance: _Optional[float] = ...) -> None: ...
    class BotSubstitution(_message.Message):
        __slots__ = ["by_team"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ...) -> None: ...
    class BotTippedOver(_message.Message):
        __slots__ = ["ball_location", "by_bot", "by_team", "location"]
        BALL_LOCATION_FIELD_NUMBER: _ClassVar[int]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        ball_location: _ssl_gc_geometry_pb2.Vector2
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., ball_location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ...) -> None: ...
    class BotTooFastInStop(_message.Message):
        __slots__ = ["by_bot", "by_team", "location", "speed"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        SPEED_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        location: _ssl_gc_geometry_pb2.Vector2
        speed: float
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., speed: _Optional[float] = ...) -> None: ...
    class BoundaryCrossing(_message.Message):
        __slots__ = ["by_team", "location"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ...) -> None: ...
    class ChallengeFlag(_message.Message):
        __slots__ = ["by_team"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ...) -> None: ...
    class ChallengeFlagHandled(_message.Message):
        __slots__ = ["accepted", "by_team"]
        ACCEPTED_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        accepted: bool
        by_team: _ssl_gc_common_pb2.Team
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., accepted: bool = ...) -> None: ...
    class ChippedGoal(_message.Message):
        __slots__ = ["by_bot", "by_team", "kick_location", "location", "max_ball_height"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        KICK_LOCATION_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        MAX_BALL_HEIGHT_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        kick_location: _ssl_gc_geometry_pb2.Vector2
        location: _ssl_gc_geometry_pb2.Vector2
        max_ball_height: float
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., kick_location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., max_ball_height: _Optional[float] = ...) -> None: ...
    class DefenderInDefenseArea(_message.Message):
        __slots__ = ["by_bot", "by_team", "distance", "location"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        DISTANCE_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        distance: float
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., distance: _Optional[float] = ...) -> None: ...
    class DefenderInDefenseAreaPartially(_message.Message):
        __slots__ = ["ball_location", "by_bot", "by_team", "distance", "location"]
        BALL_LOCATION_FIELD_NUMBER: _ClassVar[int]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        DISTANCE_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        ball_location: _ssl_gc_geometry_pb2.Vector2
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        distance: float
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., distance: _Optional[float] = ..., ball_location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ...) -> None: ...
    class DefenderTooCloseToKickPoint(_message.Message):
        __slots__ = ["by_bot", "by_team", "distance", "location"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        DISTANCE_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        distance: float
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., distance: _Optional[float] = ...) -> None: ...
    class EmergencyStop(_message.Message):
        __slots__ = ["by_team"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ...) -> None: ...
    class Goal(_message.Message):
        __slots__ = ["by_team", "kick_location", "kicking_bot", "kicking_team", "last_touch_by_team", "location", "max_ball_height", "message", "num_robots_by_team"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        KICKING_BOT_FIELD_NUMBER: _ClassVar[int]
        KICKING_TEAM_FIELD_NUMBER: _ClassVar[int]
        KICK_LOCATION_FIELD_NUMBER: _ClassVar[int]
        LAST_TOUCH_BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        MAX_BALL_HEIGHT_FIELD_NUMBER: _ClassVar[int]
        MESSAGE_FIELD_NUMBER: _ClassVar[int]
        NUM_ROBOTS_BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        kick_location: _ssl_gc_geometry_pb2.Vector2
        kicking_bot: int
        kicking_team: _ssl_gc_common_pb2.Team
        last_touch_by_team: int
        location: _ssl_gc_geometry_pb2.Vector2
        max_ball_height: float
        message: str
        num_robots_by_team: int
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., kicking_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., kicking_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., kick_location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., max_ball_height: _Optional[float] = ..., num_robots_by_team: _Optional[int] = ..., last_touch_by_team: _Optional[int] = ..., message: _Optional[str] = ...) -> None: ...
    class IndirectGoal(_message.Message):
        __slots__ = ["by_bot", "by_team", "kick_location", "location"]
        BY_BOT_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        KICK_LOCATION_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        by_bot: int
        by_team: _ssl_gc_common_pb2.Team
        kick_location: _ssl_gc_geometry_pb2.Vector2
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., by_bot: _Optional[int] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., kick_location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ...) -> None: ...
    class KeeperHeldBall(_message.Message):
        __slots__ = ["by_team", "duration", "location"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        DURATION_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        duration: float
        location: _ssl_gc_geometry_pb2.Vector2
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., duration: _Optional[float] = ...) -> None: ...
    class KickTimeout(_message.Message):
        __slots__ = ["by_team", "location", "time"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        TIME_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        location: _ssl_gc_geometry_pb2.Vector2
        time: float
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., time: _Optional[float] = ...) -> None: ...
    class MultipleCards(_message.Message):
        __slots__ = ["by_team"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ...) -> None: ...
    class MultipleFouls(_message.Message):
        __slots__ = ["by_team", "caused_game_events"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        CAUSED_GAME_EVENTS_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        caused_game_events: _containers.RepeatedCompositeFieldContainer[GameEvent]
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., caused_game_events: _Optional[_Iterable[_Union[GameEvent, _Mapping]]] = ...) -> None: ...
    class MultiplePlacementFailures(_message.Message):
        __slots__ = ["by_team"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ...) -> None: ...
    class NoProgressInGame(_message.Message):
        __slots__ = ["location", "time"]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        TIME_FIELD_NUMBER: _ClassVar[int]
        location: _ssl_gc_geometry_pb2.Vector2
        time: float
        def __init__(self, location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., time: _Optional[float] = ...) -> None: ...
    class PenaltyKickFailed(_message.Message):
        __slots__ = ["by_team", "location", "reason"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        LOCATION_FIELD_NUMBER: _ClassVar[int]
        REASON_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        location: _ssl_gc_geometry_pb2.Vector2
        reason: str
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ..., reason: _Optional[str] = ...) -> None: ...
    class PlacementFailed(_message.Message):
        __slots__ = ["by_team", "remaining_distance"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        REMAINING_DISTANCE_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        remaining_distance: float
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., remaining_distance: _Optional[float] = ...) -> None: ...
    class PlacementSucceeded(_message.Message):
        __slots__ = ["by_team", "distance", "precision", "time_taken"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        DISTANCE_FIELD_NUMBER: _ClassVar[int]
        PRECISION_FIELD_NUMBER: _ClassVar[int]
        TIME_TAKEN_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        distance: float
        precision: float
        time_taken: float
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., time_taken: _Optional[float] = ..., precision: _Optional[float] = ..., distance: _Optional[float] = ...) -> None: ...
    class Prepared(_message.Message):
        __slots__ = ["time_taken"]
        TIME_TAKEN_FIELD_NUMBER: _ClassVar[int]
        time_taken: float
        def __init__(self, time_taken: _Optional[float] = ...) -> None: ...
    class TooManyRobots(_message.Message):
        __slots__ = ["ball_location", "by_team", "num_robots_allowed", "num_robots_on_field"]
        BALL_LOCATION_FIELD_NUMBER: _ClassVar[int]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        NUM_ROBOTS_ALLOWED_FIELD_NUMBER: _ClassVar[int]
        NUM_ROBOTS_ON_FIELD_FIELD_NUMBER: _ClassVar[int]
        ball_location: _ssl_gc_geometry_pb2.Vector2
        by_team: _ssl_gc_common_pb2.Team
        num_robots_allowed: int
        num_robots_on_field: int
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., num_robots_allowed: _Optional[int] = ..., num_robots_on_field: _Optional[int] = ..., ball_location: _Optional[_Union[_ssl_gc_geometry_pb2.Vector2, _Mapping]] = ...) -> None: ...
    class UnsportingBehaviorMajor(_message.Message):
        __slots__ = ["by_team", "reason"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        REASON_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        reason: str
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., reason: _Optional[str] = ...) -> None: ...
    class UnsportingBehaviorMinor(_message.Message):
        __slots__ = ["by_team", "reason"]
        BY_TEAM_FIELD_NUMBER: _ClassVar[int]
        REASON_FIELD_NUMBER: _ClassVar[int]
        by_team: _ssl_gc_common_pb2.Team
        reason: str
        def __init__(self, by_team: _Optional[_Union[_ssl_gc_common_pb2.Team, str]] = ..., reason: _Optional[str] = ...) -> None: ...
    AIMLESS_KICK: GameEvent.Type
    AIMLESS_KICK_FIELD_NUMBER: _ClassVar[int]
    ATTACKER_DOUBLE_TOUCHED_BALL: GameEvent.Type
    ATTACKER_DOUBLE_TOUCHED_BALL_FIELD_NUMBER: _ClassVar[int]
    ATTACKER_TOO_CLOSE_TO_DEFENSE_AREA: GameEvent.Type
    ATTACKER_TOO_CLOSE_TO_DEFENSE_AREA_FIELD_NUMBER: _ClassVar[int]
    ATTACKER_TOUCHED_BALL_IN_DEFENSE_AREA: GameEvent.Type
    ATTACKER_TOUCHED_BALL_IN_DEFENSE_AREA_FIELD_NUMBER: _ClassVar[int]
    BALL_LEFT_FIELD_GOAL_LINE: GameEvent.Type
    BALL_LEFT_FIELD_GOAL_LINE_FIELD_NUMBER: _ClassVar[int]
    BALL_LEFT_FIELD_TOUCH_LINE: GameEvent.Type
    BALL_LEFT_FIELD_TOUCH_LINE_FIELD_NUMBER: _ClassVar[int]
    BOT_CRASH_DRAWN: GameEvent.Type
    BOT_CRASH_DRAWN_FIELD_NUMBER: _ClassVar[int]
    BOT_CRASH_UNIQUE: GameEvent.Type
    BOT_CRASH_UNIQUE_FIELD_NUMBER: _ClassVar[int]
    BOT_DRIBBLED_BALL_TOO_FAR: GameEvent.Type
    BOT_DRIBBLED_BALL_TOO_FAR_FIELD_NUMBER: _ClassVar[int]
    BOT_HELD_BALL_DELIBERATELY: GameEvent.Type
    BOT_HELD_BALL_DELIBERATELY_FIELD_NUMBER: _ClassVar[int]
    BOT_INTERFERED_PLACEMENT: GameEvent.Type
    BOT_INTERFERED_PLACEMENT_FIELD_NUMBER: _ClassVar[int]
    BOT_KICKED_BALL_TOO_FAST: GameEvent.Type
    BOT_KICKED_BALL_TOO_FAST_FIELD_NUMBER: _ClassVar[int]
    BOT_PUSHED_BOT: GameEvent.Type
    BOT_PUSHED_BOT_FIELD_NUMBER: _ClassVar[int]
    BOT_SUBSTITUTION: GameEvent.Type
    BOT_SUBSTITUTION_FIELD_NUMBER: _ClassVar[int]
    BOT_TIPPED_OVER: GameEvent.Type
    BOT_TIPPED_OVER_FIELD_NUMBER: _ClassVar[int]
    BOT_TOO_FAST_IN_STOP: GameEvent.Type
    BOT_TOO_FAST_IN_STOP_FIELD_NUMBER: _ClassVar[int]
    BOUNDARY_CROSSING: GameEvent.Type
    BOUNDARY_CROSSING_FIELD_NUMBER: _ClassVar[int]
    CHALLENGE_FLAG: GameEvent.Type
    CHALLENGE_FLAG_FIELD_NUMBER: _ClassVar[int]
    CHALLENGE_FLAG_HANDLED: GameEvent.Type
    CHALLENGE_FLAG_HANDLED_FIELD_NUMBER: _ClassVar[int]
    CREATED_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    DEFENDER_IN_DEFENSE_AREA: GameEvent.Type
    DEFENDER_IN_DEFENSE_AREA_FIELD_NUMBER: _ClassVar[int]
    DEFENDER_TOO_CLOSE_TO_KICK_POINT: GameEvent.Type
    DEFENDER_TOO_CLOSE_TO_KICK_POINT_FIELD_NUMBER: _ClassVar[int]
    EMERGENCY_STOP: GameEvent.Type
    EMERGENCY_STOP_FIELD_NUMBER: _ClassVar[int]
    GOAL: GameEvent.Type
    GOAL_FIELD_NUMBER: _ClassVar[int]
    INVALID_GOAL: GameEvent.Type
    INVALID_GOAL_FIELD_NUMBER: _ClassVar[int]
    KEEPER_HELD_BALL: GameEvent.Type
    KEEPER_HELD_BALL_FIELD_NUMBER: _ClassVar[int]
    MULTIPLE_CARDS: GameEvent.Type
    MULTIPLE_CARDS_FIELD_NUMBER: _ClassVar[int]
    MULTIPLE_FOULS: GameEvent.Type
    MULTIPLE_FOULS_FIELD_NUMBER: _ClassVar[int]
    NO_PROGRESS_IN_GAME: GameEvent.Type
    NO_PROGRESS_IN_GAME_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    PENALTY_KICK_FAILED: GameEvent.Type
    PENALTY_KICK_FAILED_FIELD_NUMBER: _ClassVar[int]
    PLACEMENT_FAILED: GameEvent.Type
    PLACEMENT_FAILED_FIELD_NUMBER: _ClassVar[int]
    PLACEMENT_SUCCEEDED: GameEvent.Type
    PLACEMENT_SUCCEEDED_FIELD_NUMBER: _ClassVar[int]
    POSSIBLE_GOAL: GameEvent.Type
    POSSIBLE_GOAL_FIELD_NUMBER: _ClassVar[int]
    TOO_MANY_ROBOTS: GameEvent.Type
    TOO_MANY_ROBOTS_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_GAME_EVENT_TYPE: GameEvent.Type
    UNSPORTING_BEHAVIOR_MAJOR: GameEvent.Type
    UNSPORTING_BEHAVIOR_MAJOR_FIELD_NUMBER: _ClassVar[int]
    UNSPORTING_BEHAVIOR_MINOR: GameEvent.Type
    UNSPORTING_BEHAVIOR_MINOR_FIELD_NUMBER: _ClassVar[int]
    aimless_kick: GameEvent.AimlessKick
    attacker_double_touched_ball: GameEvent.AttackerDoubleTouchedBall
    attacker_too_close_to_defense_area: GameEvent.AttackerTooCloseToDefenseArea
    attacker_touched_ball_in_defense_area: GameEvent.AttackerTouchedBallInDefenseArea
    ball_left_field_goal_line: GameEvent.BallLeftField
    ball_left_field_touch_line: GameEvent.BallLeftField
    bot_crash_drawn: GameEvent.BotCrashDrawn
    bot_crash_unique: GameEvent.BotCrashUnique
    bot_dribbled_ball_too_far: GameEvent.BotDribbledBallTooFar
    bot_held_ball_deliberately: GameEvent.BotHeldBallDeliberately
    bot_interfered_placement: GameEvent.BotInterferedPlacement
    bot_kicked_ball_too_fast: GameEvent.BotKickedBallTooFast
    bot_pushed_bot: GameEvent.BotPushedBot
    bot_substitution: GameEvent.BotSubstitution
    bot_tipped_over: GameEvent.BotTippedOver
    bot_too_fast_in_stop: GameEvent.BotTooFastInStop
    boundary_crossing: GameEvent.BoundaryCrossing
    challenge_flag: GameEvent.ChallengeFlag
    challenge_flag_handled: GameEvent.ChallengeFlagHandled
    created_timestamp: int
    defender_in_defense_area: GameEvent.DefenderInDefenseArea
    defender_too_close_to_kick_point: GameEvent.DefenderTooCloseToKickPoint
    emergency_stop: GameEvent.EmergencyStop
    goal: GameEvent.Goal
    invalid_goal: GameEvent.Goal
    keeper_held_ball: GameEvent.KeeperHeldBall
    multiple_cards: GameEvent.MultipleCards
    multiple_fouls: GameEvent.MultipleFouls
    no_progress_in_game: GameEvent.NoProgressInGame
    origin: _containers.RepeatedScalarFieldContainer[str]
    penalty_kick_failed: GameEvent.PenaltyKickFailed
    placement_failed: GameEvent.PlacementFailed
    placement_succeeded: GameEvent.PlacementSucceeded
    possible_goal: GameEvent.Goal
    too_many_robots: GameEvent.TooManyRobots
    type: GameEvent.Type
    unsporting_behavior_major: GameEvent.UnsportingBehaviorMajor
    unsporting_behavior_minor: GameEvent.UnsportingBehaviorMinor
    def __init__(self, type: _Optional[_Union[GameEvent.Type, str]] = ..., origin: _Optional[_Iterable[str]] = ..., created_timestamp: _Optional[int] = ..., ball_left_field_touch_line: _Optional[_Union[GameEvent.BallLeftField, _Mapping]] = ..., ball_left_field_goal_line: _Optional[_Union[GameEvent.BallLeftField, _Mapping]] = ..., aimless_kick: _Optional[_Union[GameEvent.AimlessKick, _Mapping]] = ..., attacker_too_close_to_defense_area: _Optional[_Union[GameEvent.AttackerTooCloseToDefenseArea, _Mapping]] = ..., defender_in_defense_area: _Optional[_Union[GameEvent.DefenderInDefenseArea, _Mapping]] = ..., boundary_crossing: _Optional[_Union[GameEvent.BoundaryCrossing, _Mapping]] = ..., keeper_held_ball: _Optional[_Union[GameEvent.KeeperHeldBall, _Mapping]] = ..., bot_dribbled_ball_too_far: _Optional[_Union[GameEvent.BotDribbledBallTooFar, _Mapping]] = ..., bot_pushed_bot: _Optional[_Union[GameEvent.BotPushedBot, _Mapping]] = ..., bot_held_ball_deliberately: _Optional[_Union[GameEvent.BotHeldBallDeliberately, _Mapping]] = ..., bot_tipped_over: _Optional[_Union[GameEvent.BotTippedOver, _Mapping]] = ..., attacker_touched_ball_in_defense_area: _Optional[_Union[GameEvent.AttackerTouchedBallInDefenseArea, _Mapping]] = ..., bot_kicked_ball_too_fast: _Optional[_Union[GameEvent.BotKickedBallTooFast, _Mapping]] = ..., bot_crash_unique: _Optional[_Union[GameEvent.BotCrashUnique, _Mapping]] = ..., bot_crash_drawn: _Optional[_Union[GameEvent.BotCrashDrawn, _Mapping]] = ..., defender_too_close_to_kick_point: _Optional[_Union[GameEvent.DefenderTooCloseToKickPoint, _Mapping]] = ..., bot_too_fast_in_stop: _Optional[_Union[GameEvent.BotTooFastInStop, _Mapping]] = ..., bot_interfered_placement: _Optional[_Union[GameEvent.BotInterferedPlacement, _Mapping]] = ..., possible_goal: _Optional[_Union[GameEvent.Goal, _Mapping]] = ..., goal: _Optional[_Union[GameEvent.Goal, _Mapping]] = ..., invalid_goal: _Optional[_Union[GameEvent.Goal, _Mapping]] = ..., attacker_double_touched_ball: _Optional[_Union[GameEvent.AttackerDoubleTouchedBall, _Mapping]] = ..., placement_succeeded: _Optional[_Union[GameEvent.PlacementSucceeded, _Mapping]] = ..., penalty_kick_failed: _Optional[_Union[GameEvent.PenaltyKickFailed, _Mapping]] = ..., no_progress_in_game: _Optional[_Union[GameEvent.NoProgressInGame, _Mapping]] = ..., placement_failed: _Optional[_Union[GameEvent.PlacementFailed, _Mapping]] = ..., multiple_cards: _Optional[_Union[GameEvent.MultipleCards, _Mapping]] = ..., multiple_fouls: _Optional[_Union[GameEvent.MultipleFouls, _Mapping]] = ..., bot_substitution: _Optional[_Union[GameEvent.BotSubstitution, _Mapping]] = ..., too_many_robots: _Optional[_Union[GameEvent.TooManyRobots, _Mapping]] = ..., challenge_flag: _Optional[_Union[GameEvent.ChallengeFlag, _Mapping]] = ..., challenge_flag_handled: _Optional[_Union[GameEvent.ChallengeFlagHandled, _Mapping]] = ..., emergency_stop: _Optional[_Union[GameEvent.EmergencyStop, _Mapping]] = ..., unsporting_behavior_minor: _Optional[_Union[GameEvent.UnsportingBehaviorMinor, _Mapping]] = ..., unsporting_behavior_major: _Optional[_Union[GameEvent.UnsportingBehaviorMajor, _Mapping]] = ...) -> None: ...
