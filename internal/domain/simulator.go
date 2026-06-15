package domain

import (
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// SimulationResult é o resultado puro de um intervalo de simulação.
// Sem side effects — facilita testes unitários.
type SimulationResult struct {
	XP             int64
	Gold           int64
	KillCounts     map[uuid.UUID]int
	LootDrops      map[uuid.UUID]LootDrop
	PhysicalHits   int
	ManaConsumed   int64
	Died           bool
	TimeUntilDeath time.Duration
	Completed      bool
}

type LootDrop struct {
	Quantity int
	Rarity   string
}

// CombatSimulator é a interface do simulador de combate.
type CombatSimulator interface {
	Resolve(session HuntSession, hunt Hunt, monsters []MonsterWithSpawnRate, delta time.Duration, totalElapsed time.Duration) SimulationResult
}

type MonsterWithSpawnRate struct {
	Monster   Monster
	SpawnRate float64
}

type combatSimulator struct{}

func NewCombatSimulator() CombatSimulator {
	return &combatSimulator{}
}

// Resolve calcula o resultado de um intervalo de hunt de forma determinística.
// CFD: AttackPower = WeaponAttack + (Skill × 2) + (Level × 0.2)
// CFD: DefensePower = Armor + ShieldDefense + (Shielding × 1.5)
func (s *combatSimulator) Resolve(
	session HuntSession,
	hunt Hunt,
	monsters []MonsterWithSpawnRate,
	delta time.Duration,
	totalElapsed time.Duration,
) SimulationResult {
	deltaMinutes := delta.Minutes()
	if deltaMinutes <= 0 {
		return SimulationResult{}
	}

	completed := totalElapsed >= time.Duration(session.ConfiguredDuration)*time.Minute

	attackPower := calcAttackPower(session)
	defPower := calcDefensePower(session)

	// Eficiência de kills baseada na build vs hunt
	// Base: hunt.xp_per_hour como referência de kills/hora no nível recomendado
	// Ajuste por attack power (simplificado — BRD define os multiplicadores exatos)
	killEfficiency := math.Min(2.0, math.Max(0.1, float64(attackPower)/100.0))
	killsPerHour := float64(hunt.XPPerHour) / 50.0 * killEfficiency // 50 xp base por kill
	killsInInterval := killsPerHour / 60.0 * deltaMinutes

	xpInInterval := int64(killsInInterval * float64(hunt.XPPerHour) / killsPerHour)
	if killsPerHour > 0 {
		xpInInterval = int64(killsInInterval * 50 * killEfficiency)
	}

	// Gold: variação ±20% sobre gold_per_hour
	goldBase := float64(hunt.GoldPerHour) / 60.0 * deltaMinutes
	goldVariance := goldBase * 0.2 * (rand.Float64()*2 - 1)
	goldInInterval := int64(math.Max(0, goldBase+goldVariance))

	// Mortalidade: probabilidade de morte no intervalo
	// mortality_rate é por hora; escala para o delta
	deathProb := hunt.MortalityRate * deltaMinutes / 60.0
	// Mitigation reduz probabilidade de morte
	mitigationFactor := math.Min(0.8, float64(defPower)/200.0)
	deathProb *= (1 - mitigationFactor)

	died := rand.Float64() < deathProb
	var timeUntilDeath time.Duration
	if died {
		// Momento da morte dentro do intervalo
		timeUntilDeath = time.Duration(rand.Float64() * float64(delta))
		// Recalcula XP e gold até o momento da morte
		fraction := float64(timeUntilDeath) / float64(delta)
		xpInInterval = int64(float64(xpInInterval) * fraction)
		goldInInterval = int64(float64(goldInInterval) * fraction)
	}

	// Kill counts por monstro
	killCounts := make(map[uuid.UUID]int, len(monsters))
	lootDrops := make(map[uuid.UUID]LootDrop)

	for _, m := range monsters {
		monsterKills := int(math.Round(killsInInterval * m.SpawnRate))
		if monsterKills <= 0 {
			continue
		}
		killCounts[m.Monster.ID] = monsterKills

		for _, entry := range m.Monster.LootTable {
			drops := int(math.Floor(float64(monsterKills) * entry.DropChance))
			if drops <= 0 {
				// Chance fracionária: rolar para cada kill individual seria caro.
				// Aproximamos: se monsterKills * dropChance >= 0.5, garante 1 drop.
				if float64(monsterKills)*entry.DropChance >= 0.5 {
					drops = 1
				} else {
					continue
				}
			}
			qty := drops * (entry.QuantityMin + rand.Intn(entry.QuantityMax-entry.QuantityMin+1))
			existing := lootDrops[entry.TemplateID]
			lootDrops[entry.TemplateID] = LootDrop{
				Quantity: existing.Quantity + qty,
				Rarity:   entry.Rarity,
			}
		}
	}

	// Physical hits e mana consumed para o Character Domain calcular skill XP
	physicalHits := int(killsInInterval)
	// Mana: depende da vocação e das skills
	manaConsumed := estimateManaConsumed(session, deltaMinutes)

	return SimulationResult{
		XP:             xpInInterval,
		Gold:           goldInInterval,
		KillCounts:     killCounts,
		LootDrops:      lootDrops,
		PhysicalHits:   physicalHits,
		ManaConsumed:   manaConsumed,
		Died:           died,
		TimeUntilDeath: timeUntilDeath,
		Completed:      completed,
	}
}

// calcAttackPower — CFD: AttackPower = WeaponAttack + (Skill × 2) + (Level × 0.2)
func calcAttackPower(session HuntSession) int {
	weaponAttack := 0
	if eq, ok := session.SnapshotEquipment["weapon"]; ok && eq != nil {
		weaponAttack = eq.Attack
	}

	skillLevel := 10 // base
	voc := session.SnapshotVocation
	switch voc {
	case VocationKnight:
		// Usa a maior skill física disponível
		for _, sk := range []string{"sword", "axe", "club"} {
			if s, ok := session.SnapshotSkills[sk]; ok && s.Level > skillLevel {
				skillLevel = s.Level
			}
		}
	case VocationPaladin:
		if s, ok := session.SnapshotSkills["distance"]; ok {
			skillLevel = s.Level
		}
	default:
		// Sorcerer/Druid: combat via magic, physicalHits é baixo
		skillLevel = 10
	}

	return weaponAttack + (skillLevel * 2) + int(float64(session.SnapshotLevel)*0.2)
}

// calcDefensePower — CFD: DefensePower = Armor + ShieldDefense + (Shielding × 1.5)
func calcDefensePower(session HuntSession) int {
	totalArmor := 0
	shieldDefense := 0
	armorSlots := []string{"helmet", "armor", "legs", "boots", "ring", "necklace"}
	for _, slot := range armorSlots {
		if eq, ok := session.SnapshotEquipment[slot]; ok && eq != nil {
			totalArmor += eq.Armor
		}
	}
	if eq, ok := session.SnapshotEquipment["shield"]; ok && eq != nil {
		shieldDefense = eq.Defense
	}

	shielding := 10
	if s, ok := session.SnapshotSkills["shielding"]; ok {
		shielding = s.Level
	}

	return totalArmor + shieldDefense + int(float64(shielding)*1.5)
}

func estimateManaConsumed(session HuntSession, deltaMinutes float64) int64 {
	// Mana consumida por minuto estimada pela vocação
	// Sorcerer/Druid consomem muito mais (magia é o ataque principal)
	// Valores serão calibrados pelo BRD
	var manaPerMinute float64
	switch session.SnapshotVocation {
	case VocationSorcerer, VocationDruid:
		ml := 0
		if s, ok := session.SnapshotSkills["magic_level"]; ok {
			ml = s.Level
		}
		manaPerMinute = float64(50 + ml*5)
	case VocationPaladin:
		ml := 0
		if s, ok := session.SnapshotSkills["magic_level"]; ok {
			ml = s.Level
		}
		manaPerMinute = float64(20 + ml*2)
	default: // Knight
		manaPerMinute = 5
	}
	return int64(manaPerMinute * deltaMinutes)
}
