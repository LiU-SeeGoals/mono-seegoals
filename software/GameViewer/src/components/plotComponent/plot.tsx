import React from 'react';
import Plot from 'react-plotly.js';

const Plotter: = ({
}) => {
  const startHeightResizer = 1000;
  const resizerWidth = 5;

  const { value: resizerValue, startResizing, isHidden } = useResizeSidebar(
    true,
    startHeightResizer
  );
  const width: number = window.innerWidth;
  const Height: number = window.innerHeight
  return (
      <Plot
        data={[
          {
            x: [1, 2, 3],
            y: [2, 6, 3],
            type: 'scatter',
            mode: 'lines+markers',
            marker: {color: 'red'},
          },
          {type: 'bar', x: [1, 2, 3], y: [2, 5, 3]},
        ]}
        layout={ {width: width, height: Height, title: {text: 'A Fancy Plot'}} }
      />
  );
};

export default Plotter;
