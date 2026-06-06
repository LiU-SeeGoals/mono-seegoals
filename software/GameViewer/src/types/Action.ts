export type Action = {
  Id: number;
  Team?: number;
  Role?: string;
  Action: number;
  PosX: number;
  PosY: number;
  PosW: number;
  DestX: number;
  DestY: number;
  DestW: number;
  Dribble: boolean;
  PreviousAction: number;
  Path?: { X: number; Y: number }[];
};
