import { useState, useCallback, useEffect } from 'react';

interface UseResizeSidebarReturn {
  value: number;
  startResizing: (mouseDownEvent: React.MouseEvent<HTMLDivElement>) => void;
  isHidden: boolean;
}

const useResizeSidebar = (
  isHorizontal: boolean,
  startValue: number
): UseResizeSidebarReturn => {
  const [value, setValue] = useState<number>(startValue);
  const [isResizing, setIsResizing] = useState<boolean>(false);
  const [previousValue, setPreviousValue] = useState<number>(startValue);
  const [isHidden, setIsHidden] = useState<boolean>(false);
  const [mouseDownTime, setMouseDownTime] = useState<number>(0);
  const [mouseDownPosition, setMouseDownPosition] = useState<{x: number, y: number}>({x: 0, y: 0});

  const startResizing = useCallback(
    (mouseDownEvent: React.MouseEvent<HTMLDivElement>) => {
      setMouseDownTime(Date.now());
      setMouseDownPosition({x: mouseDownEvent.clientX, y: mouseDownEvent.clientY});
      setIsResizing(true);
      mouseDownEvent.preventDefault();
    },
    []
  );

  const stopResizing = useCallback((mouseUpEvent: MouseEvent) => {
    if (isResizing) {
      const mouseUpTime = Date.now();
      const mouseMovement = Math.abs(mouseUpEvent.clientX - mouseDownPosition.x) + 
                           Math.abs(mouseUpEvent.clientY - mouseDownPosition.y);

      // If it was a short duration with minimal movement, consider it a click
      if (mouseUpTime - mouseDownTime < 200 && mouseMovement < 5) {
        toggleVisibility();
      }

      setIsResizing(false);
    }
  }, [isResizing, mouseDownPosition, mouseDownTime]);

  const toggleVisibility = useCallback(() => {
    if (!isHidden) {
      setPreviousValue(value);
      setIsHidden(true);
      setValue(0);
    } else {
      setValue(previousValue);
      setIsHidden(false);
      setValue(previousValue);
    }
  }, [isHidden, value, previousValue]);

  const resize = useCallback(
    (mouseMoveEvent: MouseEvent) => {
      if (!isResizing) {
        return;
      }

      const newValue: number = isHorizontal
        ? mouseMoveEvent.clientY
        : mouseMoveEvent.clientX;

      setValue(newValue);
    },
    [isResizing, isHorizontal]
  );

  useEffect(() => {
    const handleResize = (e: MouseEvent) => resize(e);
    const handleStopResizing = (e: MouseEvent) => stopResizing(e);

    if (isResizing) {
      window.addEventListener('mousemove', handleResize);
      window.addEventListener('mouseup', handleStopResizing);
    }

    return () => {
      window.removeEventListener('mousemove', handleResize);
      window.removeEventListener('mouseup', handleStopResizing);
    };
  }, [isResizing, resize, stopResizing]);

  return {
    value: value,
    startResizing,
    isHidden,
  };
};

export default useResizeSidebar;
