// What the vision server forwards from the game controller's referee packets.
export type RefereeTeam = { name: string; score: number };

export type Referee = {
  stage: string;
  command: string;
  yellow: RefereeTeam;
  blue: RefereeTeam;
};
