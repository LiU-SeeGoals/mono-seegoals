import React, { useEffect, useRef, useState } from 'react';
import Plot from 'react-plotly.js';

const MAX_POINTS = 200;
const WS_URL = 'ws://localhost:8088/signals'; // change to your Go backend URL

type Signal = {
  name: string;
  timestamps: string[];
  values: number[];
};

type SignalMap = Record<string, Signal>;

const Plotter: React.FC = () => {
  const [signals, setSignals] = useState<SignalMap>({});
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    const ws = new WebSocket(WS_URL);
    wsRef.current = ws;

    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);

    ws.onmessage = (event) => {
      try {
        const msg: { name: string; value: number; timestamp: string } = JSON.parse(event.data);

        setSignals((prev) => {
          const existing = prev[msg.name] ?? { name: msg.name, timestamps: [], values: [] };
          const timestamps = [...existing.timestamps, msg.timestamp].slice(-MAX_POINTS);
          const values = [...existing.values, msg.value].slice(-MAX_POINTS);
          return { ...prev, [msg.name]: { name: msg.name, timestamps, values } };
        });
      } catch (e) {
        console.error('Failed to parse signal:', e);
      }
    };

    return () => ws.close();
  }, []);

  const signalList = Object.values(signals);

  const overviewData = signalList.map((sig) => ({
    x: sig.timestamps,
    y: sig.values,
    type: 'scatter' as const,
    mode: 'lines' as const,
    name: sig.name,
  }));

  return (
    <div style={{ padding: 16, overflowY: 'auto', height: '100vh' }}>
      {/* Connection status */}
      <div style={{ marginBottom: 12 }}>
        <span style={{
          display: 'inline-block',
          width: 10, height: 10,
          borderRadius: '50%',
          backgroundColor: connected ? 'limegreen' : 'red',
          marginRight: 8,
        }} />
        {connected ? 'Connected' : 'Disconnected'} — {signalList.length} signal(s)
      </div>

      {/* Overview plot */}
      {signalList.length > 0 && (
        <div style={{ marginBottom: 24 }}>
          <h3 style={{ margin: '0 0 8px' }}>Overview</h3>
          <Plot
            data={overviewData}
            layout={{
              width: window.innerWidth - 48,
              height: 300,
              title: { text: 'All Signals' },
              legend: { orientation: 'h' },
              margin: { t: 40, b: 40, l: 50, r: 20 },
            }}
            useResizeHandler
          />
        </div>
      )}

      {/* Individual plots */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(500px, 1fr))', gap: 16 }}>
        {signalList.map((sig) => (
          <div key={sig.name}>
            <Plot
              data={[{
                x: sig.timestamps,
                y: sig.values,
                type: 'scatter',
                mode: 'lines+markers',
                name: sig.name,
                marker: { size: 4 },
              }]}
              layout={{
                title: { text: sig.name },
                height: 250,
                margin: { t: 40, b: 40, l: 50, r: 20 },
                xaxis: { title: { text: 'Time' } },
                yaxis: { title: { text: 'Value' } },
              }}
              useResizeHandler
              style={{ width: '100%' }}
            />
          </div>
        ))}
      </div>

      {signalList.length === 0 && (
        <div style={{ color: '#888', marginTop: 40, textAlign: 'center' }}>
          Waiting for signals...
        </div>
      )}
    </div>
  );
};

export default Plotter;
