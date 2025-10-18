import React, { useEffect, useState } from 'react';
import './FootballField.css';
import { AIRobot } from '../../../types/AIRobot';
import { actionToStr } from '../../../helper/defaultValues';

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

const withAlpha = (color: string, alpha: number): string => {
  if (color.startsWith('#')) {
    // Convert hex to RGB
    const r = color.slice(1, 3).parseInt(16);
    const g = color.slice(3, 5).parseInt(16);
    const b = color.slice(5, 7).parseInt(16);
    return `rgba(${r}, ${g}, ${b}, ${alpha})`;
  } else if (color.startsWith('rgb')) {
    // Modify existing rgb/rgba
    return color.replace(/rgb(a?)\(([^)]+)\)/, (_, a, values) => {
      const rgbValues = values.split(',').map((v: string) => v.trim());
      return `rgba(${rgbValues[0]}, ${rgbValues[1]}, ${rgbValues[2]}, ${alpha})`;
    });
  }
  return color; // Return as-is if not recognized
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
}) => {
  const minimumWidthForVertical = 810;
  const canvasRef = React.useRef<HTMLCanvasElement>(null);
  const containerRef = React.useRef<HTMLDivElement>(null);
  const [zoomLevel, setZoomLevel] = useState(1);

  // Creats a canvas for the football field image
  const canvasInit = (event: any) => {
    // Check if the canvas is initialized
    if (!canvasRef.current) {
      return;
    }

    const canvas: HTMLCanvasElement = canvasRef.current;
    canvas.width = width;
    canvas.height = height;

    draw(canvas);
  };


  function draw(canvas: HTMLCanvasElement) {
    const context: CanvasRenderingContext2D | null = canvas.getContext('2d');
    if (!context) {
      return;
    }

    // Clear the canvas
    context.clearRect(0, 0, context.canvas.width, context.canvas.height);

    context.save();
    context.translate(context.canvas.width / 2, context.canvas.height / 2);
    context.scale(zoomLevel, zoomLevel);
    context.translate(-context.canvas.width / 2, -context.canvas.height / 2);

    drawAllRobots(context);
    drawBall(context);
    drawActions(context);

    context.restore();
  }

  // Draws ball on the canvas
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
      // if (robot.robotId) {
      //     if (vectorSettingBlue[robot.robotId]) {
      //         drawArrow(context, robot, SPEED_ARROW_COLOR, SPEED_ARROW_THICKNESS);
      //     }
      // }
    });

    // These are the robots we control
    sslFieldUpdate.robotsYellow.map((robot) => {
      drawRobot(context, robot, COLOR_MAP.yellow, COLOR_MAP.black, 1.0);

      // if (robot.robotId) {
      //     if (vectorSettingYellow[robot.robotId]) {
      //         drawArrow(context, robot, SPEED_ARROW_COLOR, SPEED_ARROW_THICKNESS);
      //     }
      // }
    });
  };

  const drawActions = (context: CanvasRenderingContext2D) => {
    const actions: Action[] = robotActions;

    if (robotActions && robotActions.length > 0) {
      for (const action of robotActions) {
        const { canvasX, canvasY } = getCanvasCoordinates(
          action.Dest.X,
          action.Dest.Y,
          context
        );

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

  // Draws all robots on the canvas
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
    const canvasRadius = ROBOT_RADIUS * getScaler(context);
    const flatStartFrontAngle = (45 * Math.PI) / 180;
    const robotOrientation =
      robot.orientation !== undefined ? robot.orientation : 0;

    // Draw the "half moon" of the robot, leaving a flat front.
    context.beginPath();
    context.arc(
      canvasX,
      canvasY,
      canvasRadius,
      -flatStartFrontAngle - robotOrientation, // Offset the start angle with the robots current orientation
      flatStartFrontAngle - robotOrientation,
      true // Draw counter clockwise
    );
    context.fillStyle = withAlpha(fillColor, alpha);
    context.fill();
    context.strokeStyle = 'black';
    context.stroke();

    // Draw the flat front of the robot (added 1 degree of extra )
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

    //// TODO: ADD THESE

    // if (robot.selected) {
    //     drawCircle(context, robot, ROBOT_RADIUS/3, 'rgba(0, 0, 0, 1)');
    // }

    // Draw robot action over the robot
    // context.font = "20px Arial";
    // context.fillStyle = 'rgba(255, 0, 0, 1)';
    // context.textAlign = "center";
    // if (typeof robot.action === 'object' && robot.action !== null){
    //     const action_number: string = ActionToStr(robot.action);
    //     context.fillText(action_number, getCanvasCoordinates(robot.x, robot.y, context).canvasX, getCanvasCoordinates(robot.x, robot.y, context).canvasY - 10);
    // }
  };

  // Draw a black circle around the robot
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
    context.strokeStyle = 'rgba(0, 0, 0, 0)'; // make the border transparent
    context.fillStyle = color;
    context.fill();
    context.stroke();
  };

  // Draws the robots number id on the robot
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
    context.font = `bold ${FONT_SIZE * getScaler(context)}px Arial`;
    context.textAlign = 'center';
    context.textBaseline = 'middle';
    context.fillStyle = textColor;
    context.fillText(String(robot.robotId), canvasX, canvasY);
  };

  // Draws a arrow showing the direction of the robot
  /// TODO: When AI starts sending packets again:))))
  // const drawArrow = (context: CanvasRenderingContext2D, robot: Robot, color: string, thickness: number) => {
  //     context.save();
  //     const angle: number = Math.atan2(robot.VelY, robot.VelX) - Math.PI/2;
  //     const arrowLength: number = 10 * Math.hypot(robot.VelX, robot.VelY);

  //     // Don't draw the arrow if the velocity is too small
  //     if (arrowLength < ARROW_DRAW_MIN_SPEED_THRESHOLD) {
  //         return;
  //     }
  //     //const arrowLength: number = 100;
  //     // Calculate the starting point of the arrow (on the circle)
  //     const {canvasX: startX, canvasY: startY} = getCanvasCoordinates(robot.PosX, robot.PosY, context);

  //     // Calculate the end point of the arrow
  //     const endX: number = startX + arrowLength * Math.cos(angle);
  //     const endY: number = startY - arrowLength * Math.sin(angle);

  //     // Draw the line for the arrow
  //     context.beginPath();
  //     context.moveTo(startX, startY);
  //     context.lineTo(endX, endY);
  //     context.strokeStyle = color;
  //     context.lineWidth = thickness;
  //     context.stroke();

  //     const angle1 = angle + Math.PI / 7;
  //     const angle2 = angle - Math.PI / 7;
  //     const headX = endX + ARROW_HEAD_LENGTH * Math.cos(angle);
  //     const headY = endY - ARROW_HEAD_LENGTH * Math.sin(angle);

  //     // Draw the arrow head
  //     context.beginPath();
  //     context.moveTo(endX, endY);
  //     context.lineTo((headX - ARROW_HEAD_LENGTH * Math.cos(angle2)), (headY + ARROW_HEAD_LENGTH * Math.sin(angle2)));
  //     context.lineTo(headX, headY);
  //     context.lineTo((headX - ARROW_HEAD_LENGTH * Math.cos(angle1)), (headY + ARROW_HEAD_LENGTH * Math.sin(angle1)));
  //     context.lineTo(endX, endY);
  //     context.fillStyle = color;
  //     context.fill();
  //     context.lineWidth = thickness;
  //     context.stroke();
  //     context.restore(); // Used to only change the line thickness for the arrow and not for everything
  // };

  useEffect(() => {
    const canvas = canvasRef.current;
    if (canvas) {
      canvas.width = width;
      canvas.height = height;
      draw(canvas);
    }
  }, [sslFieldUpdate, width, height, zoomLevel]);

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
          scale(${zoomLevel})
        `,
      }}
      ref={containerRef}
    >
      <img
        src="./src/assets/football_field.svg"
        alt="canvas"
        style={{
          height: height,
          width: width,
        }}
        onLoad={canvasInit}
      />
      <canvas
        className="football-field-canvas"
        ref={canvasRef}
        style={{ height: height, width: width }}
      />
    </div>
  );
};

// Returns the coordinates of where the robot is on the canvas
function getCanvasCoordinates(
  x: number,
  y: number,
  context: CanvasRenderingContext2D
) {
  const scaler = getScaler(context);
  const canvasX = x * scaler + context.canvas.width / 2;
  const canvasY = context.canvas.height / 2 - y * scaler;
  return { canvasX, canvasY };
}

// Returns a scaler based on the canvas current size
function getScaler(context: CanvasRenderingContext2D) {
  const widthScale = context.canvas.width / REAL_WIDTH_FIELD;
  const heightScale = context.canvas.height / REAL_WIDTH_FIELD;
  return Math.min(widthScale, heightScale);
}

export default FootballField;

