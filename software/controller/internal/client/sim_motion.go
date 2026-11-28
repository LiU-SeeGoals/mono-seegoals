package client

import (
	"math"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/proto_go/simulation"
)

// SimMotionLimits bound the simulator-side motion controller.
type SimMotionLimits struct {
	MaxLinearMPS    float64
	MaxAngularRadPS float64
	// MaxLinearAccMPS2 limits how fast the commanded velocity may change;
	// zero disables the limit.
	MaxLinearAccMPS2 float64
}

// SimMotionLimitsFromConfig reads SIM_MAX_LINEAR_MPS, SIM_MAX_ANGULAR_RADPS
// and SIM_MAX_LINEAR_ACC_MPS2.
func SimMotionLimitsFromConfig() SimMotionLimits {
	sim := config.GetInstance().Sim
	return SimMotionLimits{
		MaxLinearMPS:     sim.MaxLinearMPS,
		MaxAngularRadPS:  sim.MaxAngularRadPS,
		MaxLinearAccMPS2: sim.MaxLinearAccMPS2,
	}
}

// DefaultSimMotionLimits are the limits the controller has always used.
var DefaultSimMotionLimits = SimMotionLimits{MaxLinearMPS: 1.0, MaxAngularRadPS: 3.0}

const (
	simLinearKp     = 0.001 // (m/s) per mm of position error
	simAngularKp    = 14.0  // (rad/s) per rad of heading error
	simDribblerRPM  = 100
	simAccStaleTime = 500 * time.Millisecond
)

// SimMotion stands in for the real robots' onboard controller: it turns
// actions into simulator velocity commands. Real robots receive (Pos, Dest)
// and close the loop themselves; the simulator only accepts velocities.
// It keeps the last command per robot to apply the acceleration limit, so
// use one SimMotion per team.
type SimMotion struct {
	limits SimMotionLimits
	now    func() time.Time
	last   [info.TEAM_SIZE]simVelocity
}

type simVelocity struct {
	vx, vy float64 // world frame, m/s
	at     time.Time
	set    bool
}

func NewSimMotion(limits SimMotionLimits) *SimMotion {
	return &SimMotion{limits: limits, now: time.Now}
}

// Command translates a; ok is false for actions the simulator cannot run.
func (m *SimMotion) Command(a action.Action) (*simulation.RobotCommand, bool) {
	id := uint32(a.RobotID())
	switch a := a.(type) {
	case *action.MoveTo:
		return m.moveTo(a), true
	case *action.Stop:
		m.forget(a.Id)
		return &simulation.RobotCommand{
			Id:            &id,
			MoveCommand:   localVelocity(0, 0, 0),
			KickSpeed:     f32(0),
			DribblerSpeed: f32(0),
		}, true
	case *action.Kick:
		return &simulation.RobotCommand{Id: &id, KickSpeed: f32(float32(a.KickSpeed))}, true
	case *action.Move:
		m.forget(a.Id)
		return &simulation.RobotCommand{
			Id:          &id,
			MoveCommand: localVelocity(float32(a.Direction.AtVec(0)), float32(a.Direction.AtVec(1)), 0),
		}, true
	case *action.Rotate:
		m.forget(a.Id)
		return &simulation.RobotCommand{Id: &id, MoveCommand: localVelocity(0, 0, float32(a.AngularVel))}, true
	case *action.Init, *action.MoveRemote:
		// Real-robot only: send an empty command so the robot keeps its slot.
		return &simulation.RobotCommand{Id: &id}, true
	default:
		return nil, false
	}
}

func (m *SimMotion) moveTo(mv *action.MoveTo) *simulation.RobotCommand {
	vxWorld := simLinearKp * (mv.Dest.X - mv.Pos.X)
	vyWorld := simLinearKp * (mv.Dest.Y - mv.Pos.Y)

	maxLinear := m.limits.MaxLinearMPS
	if mv.MaxLinearSpeed > 0 {
		maxLinear = math.Min(maxLinear, mv.MaxLinearSpeed)
	}
	speed := math.Hypot(vxWorld, vyWorld)
	minLinear := math.Min(maxLinear, mv.MinLinearSpeed)
	if speed > maxLinear {
		scale := maxLinear / speed
		vxWorld *= scale
		vyWorld *= scale
	} else if speed > 0 && speed < minLinear {
		// Close ball-orbit corrections request a speed floor so the
		// proportional controller does not crawl into alignment.
		scale := minLinear / speed
		vxWorld *= scale
		vyWorld *= scale
	}
	vxWorld, vyWorld = m.limitAcceleration(mv.Id, vxWorld, vyWorld)

	headingError := info.NormalizeAngleDelta(mv.Dest.Angle, mv.Pos.Angle)
	maxOmega := m.limits.MaxAngularRadPS
	cosW, sinW := math.Cos(mv.Pos.Angle), math.Sin(mv.Pos.Angle)

	id := uint32(mv.Id)
	cmd := &simulation.RobotCommand{
		Id: &id,
		MoveCommand: localVelocity(
			float32(vxWorld*cosW+vyWorld*sinW),
			float32(-vxWorld*sinW+vyWorld*cosW),
			float32(math.Max(-maxOmega, math.Min(maxOmega, simAngularKp*headingError))),
		),
		DribblerSpeed: f32(0),
	}
	if mv.Dribble {
		cmd.DribblerSpeed = f32(simDribblerRPM)
	}
	if mv.KickSpeed != 0 {
		kickSpeed := mv.SimKickSpeed
		if kickSpeed == 0 {
			kickSpeed = float32(mv.KickSpeed)
		}
		cmd.KickSpeed = f32(kickSpeed)
		if mv.KickAngle != 0 {
			cmd.KickAngle = f32(mv.KickAngle)
		}
	}
	return cmd
}

// limitAcceleration caps the change from the previous command to
// MaxLinearAccMPS2·dt and remembers the result.
func (m *SimMotion) limitAcceleration(robotID int, vx, vy float64) (float64, float64) {
	if robotID < 0 || robotID >= int(info.TEAM_SIZE) {
		return vx, vy
	}
	now := m.now()
	prev := &m.last[robotID]
	if acc := m.limits.MaxLinearAccMPS2; acc > 0 && prev.set {
		if dt := now.Sub(prev.at); dt > 0 && dt < simAccStaleTime {
			dvx, dvy := vx-prev.vx, vy-prev.vy
			maxDv := acc * dt.Seconds()
			if dv := math.Hypot(dvx, dvy); dv > maxDv {
				vx = prev.vx + dvx*maxDv/dv
				vy = prev.vy + dvy*maxDv/dv
			}
		}
	}
	*prev = simVelocity{vx: vx, vy: vy, at: now, set: true}
	return vx, vy
}

// forget marks a robot as stopped (or driven directly) for the accel limit.
func (m *SimMotion) forget(robotID int) {
	if robotID >= 0 && robotID < int(info.TEAM_SIZE) {
		m.last[robotID] = simVelocity{at: m.now(), set: true}
	}
}

func localVelocity(forward, left, angular float32) *simulation.RobotMoveCommand {
	return &simulation.RobotMoveCommand{
		Command: &simulation.RobotMoveCommand_LocalVelocity{
			LocalVelocity: &simulation.MoveLocalVelocity{Forward: &forward, Left: &left, Angular: &angular},
		},
	}
}

func f32(v float32) *float32 { return &v }
