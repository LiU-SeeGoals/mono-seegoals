import { useEffect, useState } from 'react';

// Define the type for the data you expect to receive
// Replace this with your actual data type
interface WebSocketData {
  // ... define the structure of your WebSocket data
}

const useWebSocket = (url: string) => {
  const [data, setData] = useState<any | null>(null);

  useEffect(() => {
    const socket = new WebSocket(url);

    socket.onmessage = (event) => {
      try {
        console.log('event data: ' + event.data);
        // const parsedData: any = JSON.parse(event.data);
        // setData(parsedData);
      } catch (e) {
        console.error('Error parsing message JSON', e);
      }
    };

    return () => {};
  }, []);

  return data;
};

export default useWebSocket;
