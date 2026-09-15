// Package posts owns community posts: the wall a community's editors publish
// to, the feed that gathers public communities together, and the reactions on
// both.
//
// Posts are deliberately NOT end-to-end encrypted, unlike messages. A post is
// published to everyone who can see the community — there is no pair of
// devices to encrypt between — and a feed the server cannot read is a feed it
// cannot rank. See docs/e2ee-design.md.
package posts

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("posts: not found")
	// ErrForbidden covers both "you may not post here" and "this is not your
	// post to delete".
	ErrForbidden = errors.New("posts: not permitted")
	ErrEmpty     = errors.New("posts: a post needs text or media")
	ErrTooLong   = errors.New("posts: text too long")
	// ErrNotCommunity is returned when the target is an ordinary group: a
	// group is a conversation, and its wall does not exist.
	ErrNotCommunity = errors.New("posts: this group is not a community")
)

// MaxTextLength bounds a post body. Generous compared to a chat message —
// a post is written once and read many times.
const MaxTextLength = 8000

// MaxMediaPerPost bounds one post's attachments.
const MaxMediaPerPost = 10

// Media kinds, mirroring the post_media CHECK constraint.
const (
	MediaImage = "image"
	MediaVideo = "video"
	MediaAudio = "audio"
	MediaFile  = "file"
)

func validMediaKind(k string) bool {
	switch k {
	case MediaImage, MediaVideo, MediaAudio, MediaFile:
		return true
	default:
		return false
	}
}

// ActorMeta identifies the acting user.
type ActorMeta struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	RoleLevel int
	IPHash    string
	RequestID string
}

// Post mirrors a row of posts, with its media and reaction summary attached.
type Post struct {
	ID          uuid.UUID
	CommunityID uuid.UUID
	AuthorID    uuid.UUID
	Text        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Media       []Media
}

type Media struct {
	ID          uuid.UUID
	Kind        string
	FileName    string
	MimeType    string
	SizeBytes   int64
	StoragePath string
	// Bytes holds the file inline when no object store is configured; exactly
	// one of Bytes and StoragePath is set (migration 44).
	Bytes    []byte
	Position int
}

// ReactionSummary is one emoji on one post: how many people chose it and
// whether the viewer is among them.
type ReactionSummary struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
	Mine  bool   `json:"mine"`
}

// AuthorCard is the small user card shown on a post.
type AuthorCard struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	Username    string    `json:"username"`
	AvatarURL   *string   `json:"avatarUrl"`
}

// CommunityCard says where a post came from, and whether the reader is in it.
//
// Carried on every feed item on purpose: a post whose community the reader
// cannot identify is a post they cannot act on. This is what powers "open the
// community" and "join" straight from the feed.
type CommunityCard struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	AvatarURL *string   `json:"avatarUrl"`
	IsMember  bool      `json:"isMember"`
	// JoinPolicy is "open" (join instantly) or "request" (apply and wait), so
	// the button can say which one it is before it is pressed.
	JoinPolicy string `json:"joinPolicy"`
}

// MediaDTO is a post attachment as the API returns it.
type MediaDTO struct {
	ID       uuid.UUID `json:"id"`
	Kind     string    `json:"kind"`
	FileName string    `json:"fileName"`
	MimeType string    `json:"mimeType"`
	Size     int64     `json:"sizeBytes"`
	URL      string    `json:"url"`
	Position int       `json:"position"`
}

// DTO is a post as the API returns it.
type DTO struct {
	ID        uuid.UUID         `json:"id"`
	Text      string            `json:"text"`
	CreatedAt time.Time         `json:"createdAt"`
	EditedAt  *time.Time        `json:"editedAt"`
	Author    AuthorCard        `json:"author"`
	Community CommunityCard     `json:"community"`
	Media     []MediaDTO        `json:"media"`
	Reactions []ReactionSummary `json:"reactions"`
	// CanDelete tells the client whether to offer deletion, so it does not
	// have to re-derive the rule (the server enforces it regardless).
	CanDelete bool `json:"canDelete"`
}

// Page is a cursor-paginated slice of the feed or of one community's wall.
type Page struct {
	Posts []DTO `json:"posts"`
	// NextCursor is empty when there is nothing more to load.
	NextCursor string `json:"nextCursor"`
}
