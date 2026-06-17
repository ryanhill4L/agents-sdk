// Package schedules implements Eve-style scheduled triggers: an agent can be
// woken on a cron expression with a fixed input. It includes a small,
// dependency-free cron matcher supporting the standard five fields
// (minute hour day-of-month month day-of-week) with "*", lists ("1,2"),
// ranges ("1-5"), and steps ("*/15").
package schedules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Schedule binds a cron expression to an input string.
type Schedule struct {
	Name  string `yaml:"name"`
	Cron  string `yaml:"cron"`
	Input string `yaml:"input"`

	schedule cronSchedule
}

// LoadDir loads every *.yaml file in dir as a schedule. A missing directory is
// not an error.
func LoadDir(dir string) ([]Schedule, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []Schedule
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var s Schedule
		if err := yaml.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if s.Name == "" {
			base := filepath.Base(path)
			s.Name = strings.TrimSuffix(base, filepath.Ext(base))
		}
		if err := s.compile(); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Schedule) compile() error {
	cs, err := parseCron(s.Cron)
	if err != nil {
		return err
	}
	s.schedule = cs
	return nil
}

// Matches reports whether the schedule should fire at time t (minute precision).
func (s *Schedule) Matches(t time.Time) bool {
	return s.schedule.matches(t)
}

// TriggerFunc handles a fired schedule.
type TriggerFunc func(ctx context.Context, s Schedule) error

// Run blocks, ticking once per minute, and invokes trigger for every schedule
// whose cron matches the current minute. It returns when ctx is cancelled.
func Run(ctx context.Context, scheds []Schedule, trigger TriggerFunc) error {
	for i := range scheds {
		if err := scheds[i].compile(); err != nil {
			return err
		}
	}

	// Align to the next minute boundary, then tick every minute.
	now := time.Now()
	next := now.Truncate(time.Minute).Add(time.Minute)
	timer := time.NewTimer(time.Until(next))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-timer.C:
			for _, s := range scheds {
				if s.Matches(t) {
					if err := trigger(ctx, s); err != nil {
						return err
					}
				}
			}
			next = next.Add(time.Minute)
			timer.Reset(time.Until(next))
		}
	}
}

// cronSchedule is a compiled 5-field cron expression.
type cronSchedule struct {
	minute     fieldSet
	hour       fieldSet
	dayOfMonth fieldSet
	month      fieldSet
	dayOfWeek  fieldSet
}

type fieldSet map[int]bool

func (s cronSchedule) matches(t time.Time) bool {
	dom := s.dayOfMonth[t.Day()]
	dow := s.dayOfWeek[int(t.Weekday())]

	// Standard cron semantics: when both DOM and DOW are restricted, a match on
	// either is sufficient; otherwise both must match.
	var dayMatch bool
	if s.dayOfMonth.restricted(1, 31) && s.dayOfWeek.restricted(0, 6) {
		dayMatch = dom || dow
	} else {
		dayMatch = dom && dow
	}

	return s.minute[t.Minute()] &&
		s.hour[t.Hour()] &&
		s.month[int(t.Month())] &&
		dayMatch
}

// restricted reports whether the field does not cover the full [min,max] range.
func (f fieldSet) restricted(min, max int) bool {
	return len(f) != (max - min + 1)
}

// parseCron parses a standard five-field cron expression.
func parseCron(expr string) (cronSchedule, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return cronSchedule{}, fmt.Errorf("cron must have 5 fields, got %d: %q", len(fields), expr)
	}

	minute, err := parseField(fields[0], 0, 59)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("minute: %w", err)
	}
	hour, err := parseField(fields[1], 0, 23)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("hour: %w", err)
	}
	dom, err := parseField(fields[2], 1, 31)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("day-of-month: %w", err)
	}
	month, err := parseField(fields[3], 1, 12)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("month: %w", err)
	}
	dow, err := parseField(fields[4], 0, 6)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("day-of-week: %w", err)
	}

	return cronSchedule{
		minute:     minute,
		hour:       hour,
		dayOfMonth: dom,
		month:      month,
		dayOfWeek:  dow,
	}, nil
}

// parseField parses a single cron field (lists, ranges, steps, wildcards).
func parseField(field string, min, max int) (fieldSet, error) {
	set := make(fieldSet)
	for _, part := range strings.Split(field, ",") {
		step := 1
		rangePart := part
		if slash := strings.Index(part, "/"); slash >= 0 {
			stepStr := part[slash+1:]
			rangePart = part[:slash]
			s, err := strconv.Atoi(stepStr)
			if err != nil || s <= 0 {
				return nil, fmt.Errorf("invalid step %q", stepStr)
			}
			step = s
		}

		lo, hi := min, max
		switch {
		case rangePart == "*":
			// full range
		case strings.Contains(rangePart, "-"):
			bounds := strings.SplitN(rangePart, "-", 2)
			a, err := strconv.Atoi(bounds[0])
			if err != nil {
				return nil, fmt.Errorf("invalid range start %q", bounds[0])
			}
			b, err := strconv.Atoi(bounds[1])
			if err != nil {
				return nil, fmt.Errorf("invalid range end %q", bounds[1])
			}
			lo, hi = a, b
		default:
			v, err := strconv.Atoi(rangePart)
			if err != nil {
				return nil, fmt.Errorf("invalid value %q", rangePart)
			}
			lo, hi = v, v
		}

		if lo < min || hi > max || lo > hi {
			return nil, fmt.Errorf("value out of range [%d,%d]: %q", min, max, part)
		}
		for v := lo; v <= hi; v += step {
			set[v] = true
		}
	}
	if len(set) == 0 {
		return nil, fmt.Errorf("empty field %q", field)
	}
	return set, nil
}
