import React, { useEffect, useState } from 'react';
import './FootballField.css';
import { AIRobot } from '../../../types/AIRobot';
import { actionToStr } from '../../../helper/defaultValues';
import { SSL_GeometryFieldSize } from '../../../proto/ssl_vision_geometry';

const TEAM_IDS = Object.freeze({
  YELLOW: 0,
  BLUE: 1,
});

const REAL_WIDTH_FIELD: number = 9600;
const ROBOT_RADIUS: number = 90;
const FONT_SIZE: number = 120;

const ARROW_HEAD_LENGTH: number = 5;
const SPEED_ARROW_COLOR: string = 'rgba(0, 0, 0, 1)';
const SPEED_ARROW_THICKNESS: number = 3;
const ARROW_DRAW_MIN_SPEED_THRESHOLD: number = 0.005;

const COLOR_MAP: Record<string, string> = {
  yellow: 'rgba(245, 239, 66, 1)',
  blue: 'rgb(19, 109, 253)',
  black: 'rgba(0, 0, 0, 1)',
  white: 'rgba(255, 255, 255, 1)',
};

function distance(a: { x: number; y: number }, b: { x: number; y: number }) {
  const dx = a.x - b.x;
  const dy = a.y - b.y;
  return Math.sqrt(dx * dx + dy * dy);
}

function findClosestRobot(robots: any[], posX, posY, threshold: number) {
  let closest = null;
  let minDist = Infinity;

  for (const robot of robots) {

    // console.log(robot)
    // console.log(posX,posY)
    const dist = distance(robot, {x:posX, y:posY});

    if (dist < minDist && dist < threshold) {
      minDist = dist;
      closest = robot;
    }
  }

  return closest;
}
            
function sendMouseClick(e, canvas, container, rotation, controllerSend, sslFieldUpdate, selectedRobot, setSelectedRobot){
  const coords = getWorldFromMouseEvent(e, canvas, container, rotation);

  const robotYellow = findClosestRobot(sslFieldUpdate.robotsYellow, coords.worldX, coords.worldY, 200)
  const robotBlue = findClosestRobot(sslFieldUpdate.robotsBlue, coords.worldX, coords.worldY, 200)

  
  if (robotYellow != null)
  {
    setSelectedRobot(robotYellow.robotId);
    selectedRobot = robotYellow.robotId;
  }

  const moveCommand = {"Command":"MOVE_ROBOT",
    x:parseInt(coords.worldX), 
    y:parseInt(coords.worldY), 
    Id: selectedRobot}

  console.log(moveCommand);
  controllerSend(moveCommand);
};

const withAlpha = (color: string, alpha: number): string => {
  if (color.startsWith('#')) {
    const r = parseInt(color.slice(1, 3), 16);
    const g = parseInt(color.slice(3, 5), 16);
    const b = parseInt(color.slice(5, 7), 16);
    return `rgba(${r}, ${g}, ${b}, ${alpha})`;
  } else if (color.startsWith('rgb')) {
    return color.replace(/rgb(a?)\(([^)]+)\)/, (_, a, values) => {
      const rgbValues = values.split(',').map((v: string) => v.trim());
      return `rgba(${rgbValues[0]}, ${rgbValues[1]}, ${rgbValues[2]}, ${alpha})`;
    });
  }
  return color;
};

interface FootBallFieldProps {
  height: number;
  width: number;
  sslFieldUpdate: SSLFieldUpdate;
  aiRobotUpdate: AIRobotUpdate;
  robotActions: Action[];
  errorOverlay: string;
  vectorSettingBlue: boolean[];
  vectorSettingYellow: boolean[];
  fieldGeometry: SSL_GeometryFieldSize | null;
}

const FootballField: React.FC<FootBallFieldProps> = ({
  height,
  width,
  sslFieldUpdate,
  aiRobotUpdate,
  robotActions,
  errorOverlay,
  vectorSettingBlue,
  vectorSettingYellow,
  fieldGeometry,
  controllerSend,
}) => {
  const minimumWidthForVertical = 810;
  const canvasRef = React.useRef<HTMLCanvasElement>(null);
  const containerRef = React.useRef<HTMLDivElement>(null);
  const [zoomLevel, setZoomLevel] = useState(1);
  const [selectedRobot, setSelectedRobot] = useState(0);

  const drawField = (context: CanvasRenderingContext2D, geometry: SSL_GeometryFieldSize) => {
    context.fillStyle = '#1a5f1a';
    context.fillRect(0, 0, context.canvas.width, context.canvas.height);

    geometry.fieldLines?.forEach(line => {
      drawLineSegment(context, line);
    });

    geometry.fieldArcs?.forEach(arc => {
      drawCircularArc(context, arc);
    });
  };

  const drawLineSegment = (
    context: CanvasRenderingContext2D, 
    line: any
  ) => {
    const { canvasX: x1, canvasY: y1 } = getCanvasCoordinates(line.p1.x, line.p1.y, context);
    const { canvasX: x2, canvasY: y2 } = getCanvasCoordinates(line.p2.x, line.p2.y, context);
  
    context.beginPath();
    context.moveTo(x1, y1);
    context.lineTo(x2, y2);
    context.strokeStyle = 'white';
    context.lineWidth = line.thickness * getScaler(context.canvas.width, context.canvas.height);
    context.lineCap = 'butt';
    context.stroke();
  };

  const drawCircularArc = (
    context: CanvasRenderingContext2D,
    arc: any
  ) => {
    const { canvasX, canvasY } = getCanvasCoordinates(arc.center.x, arc.center.y, context);
    const radius = arc.radius * getScaler(context.canvas.width, context.canvas.height);
  
    context.beginPath();
    context.arc(canvasX, canvasY, radius, arc.a1, arc.a2, false);
    context.strokeStyle = 'white';
    context.lineWidth = arc.thickness * getScaler(context.canvas.width, context.canvas.height);
    context.stroke();
  };

  function draw(canvas: HTMLCanvasElement) {
    const context: CanvasRenderingContext2D | null = canvas.getContext('2d');
    if (!context) {
      return;
    }

    context.clearRect(0, 0, context.canvas.width, context.canvas.height);

    context.save();
    context.translate(context.canvas.width / 2, context.canvas.height / 2);
    // context.scale(zoomLevel, zoomLevel);
    context.translate(-context.canvas.width / 2, -context.canvas.height / 2);

    if (fieldGeometry) {
      drawField(context, fieldGeometry);
    }

    drawAllRobots(context);
    drawBall(context);
    drawActions(context);

    context.restore();
  }

  const drawBall = (context: CanvasRenderingContext2D) => {
    try {
      const ball: SSLBall = sslFieldUpdate.balls[0];
      const { canvasX, canvasY } = getCanvasCoordinates(
        ball.x,
        ball.y,
        context
      );
      context.beginPath();
      context.arc(canvasX, canvasY, 5, 0, 2 * Math.PI);
      context.strokeStyle = 'rgba(0, 0, 0, 0)';
      context.fillStyle = 'orange';
      context.fill();
      context.stroke();
    } catch (e) {
      //console.error('Ball does not exist to draw');
    }
  };

  const drawAllRobots = (context: CanvasRenderingContext2D) => {
    sslFieldUpdate.robotsBlue.map((robot) => {
      drawRobot(context, robot, COLOR_MAP.blue, COLOR_MAP.white, 1.0);
    });

    sslFieldUpdate.robotsYellow.map((robot) => {
      drawRobot(context, robot, COLOR_MAP.yellow, COLOR_MAP.black, 1.0);
    });
  };

  const getRobotPositionById = (id: number): { x: number; y: number } | null => {
    const blue = sslFieldUpdate.robotsBlue.find((r) => r.robotId === id);
    if (blue) return { x: blue.x, y: blue.y };
    const yellow = sslFieldUpdate.robotsYellow.find((r) => r.robotId === id);
    if (yellow) return { x: yellow.x, y: yellow.y };
    return null;
  };

  const drawPolyline = (
    context: CanvasRenderingContext2D,
    points: { x: number; y: number }[],
    strokeStyle: string,
    lineWidth: number
  ) => {
    if (points.length < 2) return;

    const start = getCanvasCoordinates(points[0].x, points[0].y, context);
    context.beginPath();
    context.moveTo(start.canvasX, start.canvasY);
    for (let i = 1; i < points.length; i++) {
      const p = getCanvasCoordinates(points[i].x, points[i].y, context);
      context.lineTo(p.canvasX, p.canvasY);
    }
    context.strokeStyle = strokeStyle;
    context.lineWidth = lineWidth;
    context.lineJoin = 'round';
    context.lineCap = 'round';
    context.stroke();
  };

  const drawActions = (context: CanvasRenderingContext2D) => {
    const actions: Action[] = robotActions;

    if (robotActions && robotActions.length > 0) {
      for (const action of robotActions) {
        const destX = (action as any).Dest?.X ?? action.DestX;
        const destY = (action as any).Dest?.Y ?? action.DestY;
        if (destX === undefined || destY === undefined) {
          console.log("[FootballField.tsx] Got weird robot action", action);
          continue;
        }

        const { canvasX, canvasY } = getCanvasCoordinates(destX, destY, context);

        // Draw the full planned path (robot -> ...waypoints... -> goal) if present.
        // We prepend the *observed* robot position so the line starts where the robot is now.
        const robotPos = getRobotPositionById(action.Id);
        if (robotPos && action.Path && action.Path.length > 0) {
          const polyPoints = [
            robotPos,
            ...action.Path.map((p) => ({ x: p.X, y: p.Y })),
          ];
          drawPolyline(context, polyPoints, 'rgba(255, 255, 255, 0.8)', 2);
        }

        const dotColor = "black";
        context.beginPath();
        context.arc(canvasX, canvasY, 4, 0, 2 * Math.PI);
        context.fillStyle = dotColor;
        context.fill();
        context.closePath();

        context.fillStyle = "white";
        context.font = "bold 8px Arial";
        context.textAlign = "center";
        context.textBaseline = "middle";
        context.fillText(`${action.Id}`, canvasX, canvasY);

        if (action.Dribble) {
          const dribbleText = "Dribbling";
          context.fillStyle = "blue";
          context.font = "italic 10px Arial";
          context.fillText(dribbleText, canvasX, canvasY + 20);
        }
      }
    }
  };

  const drawRobot = (
    context: CanvasRenderingContext2D,
    robot: SSLRobot,
    fillColor: string,
    textColor: string,
    alpha: number
  ) => {
    const { canvasX, canvasY } = getCanvasCoordinates(
      robot.x,
      robot.y,
      context
    );
    const canvasRadius = ROBOT_RADIUS * getScaler(context.canvas.width, context.canvas.height);
    const flatStartFrontAngle = (45 * Math.PI) / 180;
    const robotOrientation =
      robot.orientation !== undefined ? robot.orientation : 0;

    context.beginPath();
    context.arc(
      canvasX,
      canvasY,
      canvasRadius,
      -flatStartFrontAngle - robotOrientation,
      flatStartFrontAngle - robotOrientation,
      true
    );
    context.fillStyle = withAlpha(fillColor, alpha);
    context.fill();
    context.strokeStyle = 'black';
    context.stroke();

    const flatFrontStartX =
      canvasX +
      canvasRadius * Math.cos(-flatStartFrontAngle - robotOrientation);
    const flatFrontStartY =
      canvasY +
      canvasRadius * Math.sin(-flatStartFrontAngle - robotOrientation);
    const flatFrontEndX =
      canvasX + canvasRadius * Math.cos(flatStartFrontAngle - robotOrientation);
    const flatFrontEndY =
      canvasY + canvasRadius * Math.sin(flatStartFrontAngle - robotOrientation);

    context.moveTo(flatFrontStartX, flatFrontStartY);
    context.lineTo(flatFrontEndX, flatFrontEndY);
    context.strokeStyle = 'black';
    context.stroke();

    drawId(context, robot, withAlpha(textColor, alpha));
  };

  const drawCircle = (
    context: CanvasRenderingContext2D,
    robot: SSLRobot,
    radius: number,
    color: string
  ) => {
    const { canvasX, canvasY } = getCanvasCoordinates(
      robot.x,
      robot.y,
      context
    );
    context.beginPath();
    context.arc(canvasX, canvasY, radius, 0, 2 * Math.PI);
    context.strokeStyle = 'rgba(0, 0, 0, 0)';
    context.fillStyle = color;
    context.fill();
    context.stroke();
  };

  const drawId = (
    context: CanvasRenderingContext2D,
    robot: SSLRobot,
    textColor: string
  ) => {
    const { canvasX, canvasY } = getCanvasCoordinates(
      robot.x,
      robot.y,
      context
    );
    context.font = `bold ${FONT_SIZE * getScaler(context.canvas.width, context.canvas.height)}px Arial`;
    context.textAlign = 'center';
    context.textBaseline = 'middle';
    context.fillStyle = textColor;
    context.fillText(String(robot.robotId), canvasX, canvasY);
  };

  useEffect(() => {
    const canvas = canvasRef.current;
    if (canvas) {
      canvas.width = width;
      canvas.height = height;
      draw(canvas);
    }
  }, [sslFieldUpdate, robotActions, width, height, zoomLevel, fieldGeometry]);

  useEffect(() => {
    const handleWheel = (event: WheelEvent) => {
      event.preventDefault();
      const newZoomLevel = zoomLevel + event.deltaY * -0.001;
      setZoomLevel(Math.max(0.5, Math.min(2, newZoomLevel)));
    };

    const canvas = canvasRef.current;
    if (canvas) {
      canvas.addEventListener('wheel', handleWheel);
      return () => {
        canvas.removeEventListener('wheel', handleWheel);
      };
    }
  }, [zoomLevel]);

  return (
    <div
      className="football-field-container"
      style={{
        height: height,
        width: width,
        transform: `
          ${width <= minimumWidthForVertical ? "rotate(90deg)" : ""}
        `,
      }}
      ref={containerRef}
    >
      <canvas
        className="football-field-canvas"
        ref={canvasRef}
        style={{ height: height, width: width }}
        tabIndex='0'
        onMouseDown={(e) =>
          sendMouseClick(
            e,
            canvasRef.current,
            containerRef.current,
            width <= minimumWidthForVertical ? 90 : 0,
            controllerSend,
            sslFieldUpdate,
            selectedRobot,
            setSelectedRobot
          )
        }
      />
    </div>
  );
};

function getCanvasCoordinates(
  x: number,
  y: number,
  context: CanvasRenderingContext2D
) {
  const scaler = getScaler(context.canvas.width, context.canvas.height);
  const canvasX = x * scaler + context.canvas.width / 2;
  const canvasY = context.canvas.height / 2 - y * scaler;
  return { canvasX, canvasY };
}

function getWorldFromMouseEvent(
  e: React.MouseEvent,
  canvas: HTMLCanvasElement,
  container: HTMLDivElement,
  rotationDeg: number,
) {
  const rect = canvas.getBoundingClientRect();

  let x = e.clientX - rect.left;
  let y = e.clientY - rect.top;

  const w = rect.width;
  const h = rect.height;

  let canvasX: number;
  let canvasY: number;

  if (rotationDeg === 90) {
    canvasX = y;
    canvasY = w - x;
  } else if (rotationDeg === -90) {
    canvasX = h - y;
    canvasY = x;
  } else {
    canvasX = x;
    canvasY = y;
  }

  const scale = getScaler(canvas.width, canvas.height);

  const worldX = (canvasX - canvas.width / 2) / scale;
  const worldY = (canvas.height / 2 - canvasY) / scale;

  return { worldX, worldY };
}

function getScaler(width, height) {
  const widthScale = width / REAL_WIDTH_FIELD;
  const heightScale = height / REAL_WIDTH_FIELD;
  return Math.min(widthScale, heightScale);
}

export default FootballField;
