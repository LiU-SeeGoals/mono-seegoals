package scripts

import (
	"fmt"
	"time"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/scripting"
)
const pause = time.Second
const reached = 500.0
const react = 150.0

// Byt dessa för att testa olika
const IdToTry = 6
const InSim = true
func TryStates(s *scripting.Script) {
	if InSim == true{
		ball := info.Position{X:-2000, Y:-2000}
		s.PlaceBall(ball)
		s.Place(2, info.Position{X: 1000, Y: 0})
		s.Place(3, info.Position{X: 0, Y: 0})
		s.Place(4, info.Position{X: 400, Y: -100})
		s.Place(5, info.Position{X: 2500, Y: 200})
		s.Place(1, info.Position{X: 830, Y: -100})
		s.Place(6, info.Position{X: 2500, Y: 2000})
		GetToBall(s,IdToTry)
		TestStopReact(s,IdToTry)
		TestWhenSurronded(s,IdToTry)
	}else
	{
		GetToBall(s,IdToTry)
		TestStopReact2(s,IdToTry)
		TestWhenSurronded2(s,IdToTry)}
	}


// func change_state(s *scripting.Script) info.RefState {
// 	return s.Game().Status.GetGameEvent().GetCurrentState()///OBs gör ej just nu ändra sen
// }
func GetToPos(s *scripting.Script,id int,pos info.Position ){
	//Tar in en id nummer och en position och kör tills den kommer till punkten-reached//
	IDE := info.ID(id)
	fmt.Printf("inne")
	s.Do(act.NewMoveToPosition(s.Team(), IDE, pos))
	for {
		RoboPos,ok := s.Robot(IDE)
			if ok && RoboPos.Dist2d(pos) <=reached{

			break}
		s.Wait(pause)
	}
	return}
func GetToBall(s *scripting.Script, id int){
// Testar bara som GetToPos men går till bollen. Denna flyttar på bollen under sim så fungerar inte i
//verkligeheten
	IDE := info.ID(id)
	fmt.Printf("inne")
	i := 0
	for {
		BallPos,ok := s.Ball()
		if !ok{
			fmt.Printf("FEL")
		}
		s.Do(act.NewMoveToPosition(s.Team(), IDE, BallPos))
		RoboPos,ok := s.Robot(IDE)
			if ok && RoboPos.Dist2d(BallPos) <=reached{
			break}
		s.Wait(pause)
		i += 1
		if i == 10{
			s.PlaceBall(info.Position{X:-2000, Y:2000})
		}
	}
	return
}

func TestStopReact(s *scripting.Script,id int){
	//Kollar hur den reagerar när bollen kommer nära
	IDE := info.ID(id)
	fmt.Printf("inne")

	for {
		RoboPos,ok := s.Robot(IDE)
		BallPos,ok2 := s.Ball()

			if ok2 && ok && RoboPos.Dist2d(BallPos) <=react{ 
			break}
		s.Wait(pause)
		s.PlaceBall(RoboPos)
		}
	
	for {
		RoboPos,ok := s.Robot(IDE)
		BallPos,ok2 := s.Ball()
			if ok2&&ok && RoboPos.Dist2d(BallPos) >=react+300{ 
			break}
		s.Wait(pause)}
	return
	}

func TestWhenSurronded(s *scripting.Script,id int){
	//När bollen är nära under stop och omringad av robotar
	IDE := info.ID(id)
	fmt.Printf("inne")
	for {
		RoboPos,ok := s.Robot(IDE)
		BallPos,ok2 := s.Ball()

			if ok2 && ok && RoboPos.Dist2d(BallPos) <=react{ 
			break}
		s.Wait(pause)
		x := RoboPos.X
		y := RoboPos.Y
		s.Place(4, info.Position{X:x+400,Y:y+320})
		s.Place(5, info.Position{X:x+400,Y:y-320})
		s.Place(1, info.Position{X:x,Y:y+400})
		s.Place(6, info.Position{X:x,Y:y-400})
		s.Place(2, info.Position{X:x-300,Y:y})
		s.PlaceBall(RoboPos)

		}
	
	for {
		RoboPos,ok := s.Robot(IDE)
		BallPos,ok2 := s.Ball()
			if ok2&&ok && RoboPos.Dist2d(BallPos) >=react+300{ 
			break}
		s.Wait(pause)}
	return
	}

func TestStopReact2(s *scripting.Script,id int){
	//Kollar hur den reagerar när bollen kommer nära
	IDE := info.ID(id)
	fmt.Printf("inne")
	i := 0
	for {
		RoboPos,ok := s.Robot(IDE)
		BallPos,ok2 := s.Ball()
		fmt.Println("not done react2")
			if ok2 && ok && RoboPos.Dist2d(BallPos) <=react{ 
			break}
		i+= 1
		if i >= 10{
			s.PlaceBall(RoboPos)
		}
		s.Wait(pause)

		}
	
	for {
		RoboPos,ok := s.Robot(IDE)
		BallPos,ok2 := s.Ball()
			if ok2&&ok && RoboPos.Dist2d(BallPos) >=react+300{ 
			break}
		s.Wait(pause)}
	return
	}

func TestWhenSurronded2(s *scripting.Script,id int){
	//När bollen är nära under stop och omringad av robotar
	IDE := info.ID(id)
	RoboPos,ok := s.Robot(IDE)
	if !ok{
		fmt.Printf("DÅLIGT")
	}
	fmt.Printf("inne")
	x := RoboPos.X
	y := RoboPos.Y
	one :=info.Position{X:x,Y:y+400}
	two := info.Position{X:x-300,Y:y}
	four:=info.Position{X:x+400,Y:y+320}
	five := info.Position{X:x+400,Y:y-320}
	sixth := info.Position{X:x,Y:y-400}
	s.Do(act.NewMoveToPosition(s.Team(), info.ID(1),one ))
	s.Do(act.NewMoveToPosition(s.Team(), info.ID(2), two))
	s.Do(act.NewMoveToPosition(s.Team(), info.ID(4), four))
	s.Do(act.NewMoveToPosition(s.Team(), info.ID(5), five))
	s.Do(act.NewMoveToPosition(s.Team(), info.ID(6), sixth))
	i := 0
	for {
		RoboPos,ok := s.Robot(IDE)
		BallPos,ok2 := s.Ball()

			if ok2 && ok && RoboPos.Dist2d(BallPos) <=react{ 
			break}
		s.Wait(pause)
		i+= 1
		if i == 10{
			s.PlaceBall(RoboPos)
		}
		}
	
	for {
		RoboPos,ok := s.Robot(IDE)
		BallPos,ok2 := s.Ball()
			if ok2&&ok && RoboPos.Dist2d(BallPos) >=react+300{ 
			break}
		s.Wait(pause)}
	return
	}
// jag ska göra så att vi kan testa hur scriptet fungerar med olika states alltså se problem med vår  state logik