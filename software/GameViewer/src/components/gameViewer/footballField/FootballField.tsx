import React, { useCallback, useEffect, useRef } from 'react';
import './FootballField.css';
import { SSLGeometryFieldSize } from '../../../proto/ssl_vision/ssl_vision_geometry';
import { applyVisionMask } from '../../../helper/visionMask';
import { actionsStore, fieldStore, rolesStore, visionMaskStore } from '../../../state/stores';
import { SSLFieldUpdate } from '../../../types/SSLFieldUpdate';
import { SSLRobot } from '../../../types/SSLRobot';
import { VisionMask } from '../../../types/VisionMask';
import { Action } from '../../../types/Action';

const TEAM_IDS = Object.freeze({
  YELLOW: 1,
  BLUE: 2,
});

const REAL_WIDTH_FIELD = 9600;
const ROBOT_RADIUS = 90;
const FONT_SIZE = 120;
const ROLE_FONT_SIZE = 85;
const MIN_WIDTH_FOR_HORIZONTAL = 810;
const CLICK_SELECT_RADIUS = 200;

const COLOR_MAP = {
  yellow: 'rgba(245, 239, 66, 1)',
  blue: 'rgb(19, 109, 253)',
  black: 'rgba(0, 0, 0, 1)',
  white: 'rgba(255, 255, 255, 1)',
};

type Point = { x: number; y: number };

interface FootballFieldProps {
  width: number;
  height: number;
  fieldGeometry: SSLGeometryFieldSize | null;
  controllerSend: (data: unknown) => void;
}

// Draws the field on a canvas. Live data (vision, actions, roles, mask) is
// read from the stores and drawn on the next animation frame, at most once
// per frame; React only re-renders this component on resize or new geometry.
const FootballField: React.FC<FootballFieldProps> = ({ width, height, fieldGeometry, controllerSend }) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const geometryRef = useRef(fieldGeometry);
  const frameRef = useRef<number | null>(null);
  const selectedRobotRef = useRef(0);
  const rotation = width <= MIN_WIDTH_FOR_HORIZONTAL ? 90 : 0;

  const requestDraw = useCallback(() => {
    if (frameRef.current !== null) return;
    frameRef.current = requestAnimationFrame(() => {
      frameRef.current = null;
      const context = canvasRef.current?.getContext('2d');
      if (context) drawScene(context, geometryRef.current);
    });
  }, []);

  useEffect(() => {
    const unsubscribe = [fieldStore, actionsStore, rolesStore, visionMaskStore]
      .map((store) => store.subscribe(requestDraw));
    return () => {
      unsubscribe.forEach((fn) => fn());
      if (frameRef.current !== null) cancelAnimationFrame(frameRef.current);
      frameRef.current = null;
    };
  }, [requestDraw]);

  // Resizing a canvas reallocates its buffer, so only do it on a real resize.
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    if (canvas.width !== width) canvas.width = width;
    if (canvas.height !== height) canvas.height = height;
    requestDraw();
  }, [width, height, requestDraw]);

  useEffect(() => {
    geometryRef.current = fieldGeometry;
    requestDraw();
  }, [fieldGeometry, requestDraw]);

  const onMouseDown = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const world = getWorldFromMouseEvent(e, canvas, rotation);
    const field = visibleField();
    const robot = findClosestRobot(field.robotsYellow, world, CLICK_SELECT_RADIUS);
    if (robot?.robotId !== undefined) selectedRobotRef.current = robot.robotId;
    controllerSend({
      Command: 'MOVE_ROBOT',
      x: Math.round(world.x),
      y: Math.round(world.y),
      Id: selectedRobotRef.current,
    });
  };

  return (
    <div
      className="football-field-container"
      style={{ height, width, transform: rotation ? `rotate(${rotation}deg)` : undefined }}
    >
      <canvas
        className="football-field-canvas"
        ref={canvasRef}
        style={{ height, width }}
        tabIndex={0}
        onMouseDown={onMouseDown}
      />
    </div>
  );
};

function visibleField(): SSLFieldUpdate {
  return applyVisionMask(fieldStore.get(), visionMaskStore.get());
}

function drawScene(context: CanvasRenderingContext2D, geometry: SSLGeometryFieldSize | null) {
  const field = visibleField();
  context.clearRect(0, 0, context.canvas.width, context.canvas.height);
  if (geometry) drawField(context, geometry);
  drawVisionMask(context, visionMaskStore.get());
  const roles = rolesStore.get();
  for (const robot of field.robotsBlue) {
    drawRobot(context, robot, TEAM_IDS.BLUE, COLOR_MAP.blue, COLOR_MAP.white, roles);
  }
  for (const robot of field.robotsYellow) {
    drawRobot(context, robot, TEAM_IDS.YELLOW, COLOR_MAP.yellow, COLOR_MAP.black, roles);
  }
  drawBall(context, field);
  drawActions(context, actionsStore.get(), field);
}

function drawField(context: CanvasRenderingContext2D, geometry: SSLGeometryFieldSize) {
  const scaler = getScaler(context.canvas.width, context.canvas.height);
  context.fillStyle = '#1a5f1a';
  context.fillRect(0, 0, context.canvas.width, context.canvas.height);
  context.strokeStyle = 'white';
  context.lineCap = 'butt';

  for (const line of geometry.fieldLines ?? []) {
    if (!line.p1 || !line.p2) continue;
    const a = toCanvas(line.p1.x, line.p1.y, context);
    const b = toCanvas(line.p2.x, line.p2.y, context);
    context.beginPath();
    context.moveTo(a.x, a.y);
    context.lineTo(b.x, b.y);
    context.lineWidth = line.thickness * scaler;
    context.stroke();
  }

  for (const arc of geometry.fieldArcs ?? []) {
    if (!arc.center) continue;
    const c = toCanvas(arc.center.x, arc.center.y, context);
    context.beginPath();
    context.arc(c.x, c.y, arc.radius * scaler, arc.a1, arc.a2, false);
    context.lineWidth = arc.thickness * scaler;
    context.stroke();
  }
}

function drawVisionMask(context: CanvasRenderingContext2D, mask: VisionMask | null) {
  if (!mask) return;
  const edgeX = toCanvas(mask.MaskedSign * mask.MarginMM, 0, context).x;
  const left = mask.MaskedSign > 0 ? edgeX : 0;
  const right = mask.MaskedSign > 0 ? context.canvas.width : edgeX;

  context.fillStyle = 'rgba(0, 0, 0, 0.55)';
  context.fillRect(left, 0, right - left, context.canvas.height);

  const scaler = getScaler(context.canvas.width, context.canvas.height);
  context.font = `bold ${Math.max(12, FONT_SIZE * 1.5 * scaler)}px Arial`;
  context.textAlign = 'center';
  context.textBaseline = 'middle';
  context.fillStyle = 'rgba(255, 255, 255, 0.7)';
  context.fillText('Vision masked', (left + right) / 2, context.canvas.height / 2);
}

function drawBall(context: CanvasRenderingContext2D, field: SSLFieldUpdate) {
  const ball = field.balls[0];
  if (!ball) return;
  const p = toCanvas(ball.x, ball.y, context);
  context.beginPath();
  context.arc(p.x, p.y, 5, 0, 2 * Math.PI);
  context.fillStyle = 'orange';
  context.fill();
}

function findRobot(field: SSLFieldUpdate, id: number, team?: number): Point | null {
  const teams = team === TEAM_IDS.BLUE ? [field.robotsBlue]
    : team === TEAM_IDS.YELLOW ? [field.robotsYellow]
      : [field.robotsBlue, field.robotsYellow];
  for (const robots of teams) {
    const robot = robots.find((r) => r.robotId === id);
    if (robot) return { x: robot.x, y: robot.y };
  }
  return null;
}

function drawPolyline(context: CanvasRenderingContext2D, points: Point[], strokeStyle: string, lineWidth: number) {
  if (points.length < 2) return;
  context.beginPath();
  points.forEach((point, i) => {
    const p = toCanvas(point.x, point.y, context);
    if (i === 0) context.moveTo(p.x, p.y);
    else context.lineTo(p.x, p.y);
  });
  context.strokeStyle = strokeStyle;
  context.lineWidth = lineWidth;
  context.lineJoin = 'round';
  context.lineCap = 'round';
  context.stroke();
}

function drawActions(context: CanvasRenderingContext2D, actions: Action[], field: SSLFieldUpdate) {
  for (const action of actions) {
    if (action.DestX === undefined || action.DestY === undefined) continue;
    const dest = toCanvas(action.DestX, action.DestY, context);

    // Planned path, starting where the robot is observed now.
    const robotPos = findRobot(field, action.Id, action.Team);
    if (robotPos && action.Path && action.Path.length > 0) {
      drawPolyline(context, [robotPos, ...action.Path.map((p) => ({ x: p.X, y: p.Y }))],
        'rgba(255, 255, 255, 0.8)', 2);
    }

    context.beginPath();
    context.arc(dest.x, dest.y, 4, 0, 2 * Math.PI);
    context.fillStyle = 'black';
    context.fill();

    context.fillStyle = 'white';
    context.font = 'bold 8px Arial';
    context.textAlign = 'center';
    context.textBaseline = 'middle';
    context.fillText(`${action.Id}`, dest.x, dest.y);

    if (action.Dribble) {
      context.fillStyle = 'blue';
      context.font = 'italic 10px Arial';
      context.fillText('Dribbling', dest.x, dest.y + 20);
    }
  }
}

function drawRobot(
  context: CanvasRenderingContext2D,
  robot: SSLRobot,
  team: number,
  fillColor: string,
  textColor: string,
  roles: Record<string, string>
) {
  const p = toCanvas(robot.x, robot.y, context);
  const scaler = getScaler(context.canvas.width, context.canvas.height);
  const radius = ROBOT_RADIUS * scaler;
  const flatFront = (45 * Math.PI) / 180;
  const orientation = robot.orientation ?? 0;

  // Body: a circle with the front flattened.
  context.beginPath();
  context.arc(p.x, p.y, radius, -flatFront - orientation, flatFront - orientation, true);
  context.closePath();
  context.fillStyle = fillColor;
  context.fill();
  context.strokeStyle = 'black';
  context.lineWidth = 1;
  context.stroke();

  context.font = `bold ${FONT_SIZE * scaler}px Arial`;
  context.textAlign = 'center';
  context.textBaseline = 'middle';
  context.fillStyle = textColor;
  context.fillText(String(robot.robotId), p.x, p.y);

  const role = roles[`${team}:${robot.robotId}`];
  if (role) {
    context.font = `bold ${Math.max(9, ROLE_FONT_SIZE * scaler)}px Arial`;
    context.textBaseline = 'bottom';
    context.lineWidth = 3;
    context.strokeStyle = 'rgba(0, 0, 0, 0.8)';
    context.fillStyle = 'white';
    context.strokeText(role, p.x, p.y - radius - 6);
    context.fillText(role, p.x, p.y - radius - 6);
  }
}

function findClosestRobot(robots: SSLRobot[], target: Point, threshold: number): SSLRobot | null {
  let closest: SSLRobot | null = null;
  let minDist = threshold;
  for (const robot of robots) {
    const dist = Math.hypot(robot.x - target.x, robot.y - target.y);
    if (dist < minDist) {
      minDist = dist;
      closest = robot;
    }
  }
  return closest;
}

function toCanvas(x: number, y: number, context: CanvasRenderingContext2D): Point {
  const scaler = getScaler(context.canvas.width, context.canvas.height);
  return {
    x: x * scaler + context.canvas.width / 2,
    y: context.canvas.height / 2 - y * scaler,
  };
}

function getWorldFromMouseEvent(e: React.MouseEvent, canvas: HTMLCanvasElement, rotationDeg: number): Point {
  const rect = canvas.getBoundingClientRect();
  const x = e.clientX - rect.left;
  const y = e.clientY - rect.top;
  // Undo the container's CSS rotation.
  const canvasX = rotationDeg === 90 ? y : x;
  const canvasY = rotationDeg === 90 ? rect.width - x : y;
  const scale = getScaler(canvas.width, canvas.height);
  return {
    x: (canvasX - canvas.width / 2) / scale,
    y: (canvas.height / 2 - canvasY) / scale,
  };
}

function getScaler(width: number, height: number) {
  return Math.min(width / REAL_WIDTH_FIELD, height / REAL_WIDTH_FIELD);
}

export default FootballField;
