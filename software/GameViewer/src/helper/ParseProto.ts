// Import the types
import { SSLFieldUpdate } from '../types/SSLFieldUpdate';

import { SSLWrapperPacket } from '../proto/ssl_wrapper';

export function parseProto(
  input_data: Uint8Array, // Change the input type to Uint8Array for binary data
  setSSLFieldUpdate: React.Dispatch<React.SetStateAction<SSLFieldUpdate>>,
  setErrorOverlay: React.Dispatch<React.SetStateAction<string>>
): void {
  try {
    // Parse the binary data into a protobuf message
    const ssl_protobuf_msg = SSLWrapperPacket.decode(input_data);
    const ssl_detections = ssl_protobuf_msg.detection;

    // console.log(input_data);
    if (ssl_detections) {
      // console.log('Received Data:', ssl_protobuf_msg); // If we want to print the message for debugging
      // Now we update all the useState varibles with the recieved json
      setSSLFieldUpdate(ssl_detections);
    }
  } catch (e) {
    setErrorOverlay('Error parsing protobuf message from SSL');
  }
}
