import { SSLFieldUpdate } from '../types/SSLFieldUpdate';
import { SSLWrapperPacket } from '../proto/ssl_wrapper';

export function parseProto(
  input_data: Uint8Array,
  setSSLFieldUpdate: React.Dispatch<React.SetStateAction<SSLFieldUpdate>>,
  setErrorOverlay: React.Dispatch<React.SetStateAction<string>>,
  setFieldGeometry?: React.Dispatch<React.SetStateAction<SSL_GeometryFieldSize | null>> // Add this parameter
): void {
  try {
    const ssl_protobuf_msg = SSLWrapperPacket.decode(input_data);

    const ssl_detections = ssl_protobuf_msg.detection;
    if (ssl_detections) {
      setSSLFieldUpdate(ssl_detections);
    }

    const ssl_geometry = ssl_protobuf_msg.geometry;
    if (ssl_geometry) {
      if (setFieldGeometry) {
        setFieldGeometry(ssl_geometry.field);
      }
    }
  } catch (e) {
    setErrorOverlay('Error parsing protobuf message from SSL');
  }
}
