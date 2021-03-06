package osu

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/wieku/danser-go/beatmap/objects"
	"github.com/wieku/danser-go/bmath/difficulty"
	"github.com/wieku/danser-go/render/batches"
	"math"
)

type Renderable interface {
	Draw(time int64, color mgl32.Vec4, batch *batches.SpriteBatch)
	DrawApproach(time int64, color mgl32.Vec4, batch *batches.SpriteBatch)
}

type objstate struct {
	buttons  buttonState
	finished bool
}

type Circle struct {
	ruleSet           *OsuRuleSet
	hitCircle         *objects.Circle
	players           []*difficultyPlayer
	state             []objstate
	fadeStartRelative float64
	renderable        *HitCircleSprite
}

func (circle *Circle) Init(ruleSet *OsuRuleSet, object objects.BaseObject, players []*difficultyPlayer) {
	circle.ruleSet = ruleSet
	circle.hitCircle = object.(*objects.Circle)
	circle.players = players
	if len(players) > 1 {
		circle.renderable = NewHitCircleSprite(*difficulty.NewDifficulty(players[0].diff.GetHPDrain(), players[0].diff.GetCS(), players[0].diff.GetOD(), players[0].diff.GetAR()), object.GetBasicData().StartPos, object.GetBasicData().StartTime)
	} else {
		circle.renderable = NewHitCircleSprite(*players[0].diff, object.GetBasicData().StartPos, object.GetBasicData().StartTime)
	}

	circle.state = make([]objstate, len(players))

	circle.fadeStartRelative = 1000000
	for _, player := range circle.players {
		circle.fadeStartRelative = math.Min(circle.fadeStartRelative, player.diff.Preempt)
	}
}

func (circle *Circle) Update(time int64) bool {
	unfinished := 0
	for i, player := range circle.players {
		state := &circle.state[i]

		if !state.finished {
			unfinished++

			if player.cursorLock == -1 {
				state.buttons.Left = player.cursor.LeftButton
				state.buttons.Right = player.cursor.RightButton
			}

			xOffset := 0.0
			yOffset := 0.0
			if player.diff.Mods&difficulty.HardRock > 0 {
				data := circle.hitCircle.GetBasicData()
				xOffset = data.StackOffset.X + float64(data.StackIndex)*player.diff.CircleRadius/(10)
				yOffset = data.StackOffset.Y - float64(data.StackIndex)*player.diff.CircleRadius/(10)
			}

			if player.cursorLock == -1 || player.cursorLock == circle.hitCircle.GetBasicData().Number {
				clicked := player.DoubleClick || (!state.buttons.Left && player.cursor.LeftButton) || (!state.buttons.Right && player.cursor.RightButton)

				if player.DoubleClick {
					player.DoubleClick = false
				} else if (!state.buttons.Left && player.cursor.LeftButton) && (!state.buttons.Right && player.cursor.RightButton) {
					player.DoubleClick = true
				}

				if clicked && player.cursor.Position.Dst(circle.hitCircle.GetPosition().SubS(xOffset, yOffset)) <= player.diff.CircleRadius {
					hit := HitResults.Miss

					relative := int64(math.Abs(float64(time - circle.hitCircle.GetBasicData().EndTime)))
					if relative < player.diff.Hit300 {
						hit = HitResults.Hit300
					} else if relative < player.diff.Hit100 {
						hit = HitResults.Hit100
					} else if relative < player.diff.Hit50 {
						hit = HitResults.Hit50
					} else if relative >= Shake {
						hit = HitResults.Ignore
					}

					if hit != HitResults.Ignore {
						combo := ComboResults.Increase
						if hit == HitResults.Miss {
							combo = ComboResults.Reset
							if len(circle.players) == 1 {
								circle.renderable.Miss(time)
							}
						} else {
							if len(circle.players) == 1 {
								circle.hitCircle.PlaySound()
								circle.renderable.Hit(time)
							}
						}

						circle.ruleSet.SendResult(time, player.cursor, circle.hitCircle.GetBasicData().Number, circle.hitCircle.GetPosition().X, circle.hitCircle.GetPosition().Y, hit, false, combo)

						player.cursorLock = -1
						state.finished = true
						continue
					}
				}

				player.cursorLock = circle.hitCircle.GetBasicData().Number
			}

			if time > circle.hitCircle.GetBasicData().EndTime+player.diff.Hit50 {
				circle.ruleSet.SendResult(time, player.cursor, circle.hitCircle.GetBasicData().Number, circle.hitCircle.GetPosition().X, circle.hitCircle.GetPosition().Y, HitResults.Miss, false, ComboResults.Reset)
				if len(circle.players) == 1 {
					circle.renderable.Miss(time)
				}
				player.cursorLock = -1
				state.finished = true
				continue
			}

			if player.cursorLock == circle.hitCircle.GetBasicData().Number {
				state.buttons.Left = player.cursor.LeftButton
				state.buttons.Right = player.cursor.RightButton
			}
		}

	}

	if len(circle.players) > 1 && time == circle.hitCircle.GetBasicData().StartTime {
		//circle.hitCircle.PlaySound()
		circle.renderable.Hit(time)
	}

	return unfinished == 0
}

func (circle *Circle) GetFadeTime() int64 {
	return circle.hitCircle.GetBasicData().StartTime - int64(circle.fadeStartRelative)
}

func (self *Circle) Draw(time int64, color mgl32.Vec4, batch *batches.SpriteBatch) {
	self.renderable.Draw(time, color, batch)
}

func (self *Circle) DrawApproach(time int64, color mgl32.Vec4, batch *batches.SpriteBatch) {
	self.renderable.DrawApproach(time, color, batch)
}
