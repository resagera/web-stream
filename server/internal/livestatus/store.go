package livestatus

import (
	"sync"
	"time"
)

type Participant struct {
	SID       string           `json:"sid"`
	Identity  string           `json:"identity"`
	Name      string           `json:"name,omitempty"`
	Tracks    map[string]Track `json:"tracks"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type Track struct {
	SID       string    `json:"sid"`
	Type      string    `json:"type,omitempty"`
	Source    string    `json:"source,omitempty"`
	Name      string    `json:"name,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Store struct {
	mu           sync.RWMutex
	participants map[string]Participant
}

func NewStore() *Store {
	return &Store{participants: make(map[string]Participant)}
}

func (s *Store) ParticipantJoined(sid, identity, name string) {
	if sid == "" {
		sid = identity
	}
	if sid == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	participant := s.participants[sid]
	if participant.Tracks == nil {
		participant.Tracks = make(map[string]Track)
	}
	participant.SID = sid
	participant.Identity = identity
	participant.Name = name
	participant.UpdatedAt = time.Now().UTC()
	s.participants[sid] = participant
}

func (s *Store) ParticipantLeft(sid, identity string) {
	if sid == "" {
		sid = identity
	}
	if sid == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.participants, sid)
}

func (s *Store) TrackPublished(participantSID, identity string, track Track) {
	if participantSID == "" {
		participantSID = identity
	}
	if participantSID == "" || track.SID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	participant := s.participants[participantSID]
	if participant.Tracks == nil {
		participant.Tracks = make(map[string]Track)
	}
	participant.SID = participantSID
	if participant.Identity == "" {
		participant.Identity = identity
	}
	track.UpdatedAt = time.Now().UTC()
	participant.Tracks[track.SID] = track
	participant.UpdatedAt = track.UpdatedAt
	s.participants[participantSID] = participant
}

func (s *Store) TrackUnpublished(participantSID, identity, trackSID string) {
	if participantSID == "" {
		participantSID = identity
	}
	if participantSID == "" || trackSID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	participant := s.participants[participantSID]
	delete(participant.Tracks, trackSID)
	participant.UpdatedAt = time.Now().UTC()
	if len(participant.Tracks) == 0 {
		delete(s.participants, participantSID)
		return
	}
	s.participants[participantSID] = participant
}

func (s *Store) Cameras() []Participant {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cameras := make([]Participant, 0, len(s.participants))
	for _, participant := range s.participants {
		if hasVideoTrack(participant) {
			cameras = append(cameras, cloneParticipant(participant))
		}
	}
	return cameras
}

func hasVideoTrack(participant Participant) bool {
	for _, track := range participant.Tracks {
		if track.Type == "video" || track.Source == "camera" || track.Source == "screen_share" {
			return true
		}
	}
	return false
}

func cloneParticipant(participant Participant) Participant {
	tracks := participant.Tracks
	participant.Tracks = map[string]Track{}
	for id, track := range tracks {
		participant.Tracks[id] = track
	}
	return participant
}
