package usecase

import (
	"context"
	"strings"
	"time"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
)

func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}

// DefaultJoinedRoomsLimit は参加ルーム一覧の省略時 limit。
const DefaultJoinedRoomsLimit = 200

type RoomUseCase struct {
	Tx       repository.RoomUnitOfWork
	Users    repository.UserRepository
	Rooms    repository.RoomRepository
	Members  repository.RoomMemberRepository
	Messages repository.MessageRepository
	Reads    repository.MessageReadRepository
	Online   UserOnlineLookup
}

// NewRoomUseCase は RoomUseCase を組み立てる。Tx はルームとメンバーの一括書き込み用。Online は省略可。
func NewRoomUseCase(tx repository.RoomUnitOfWork, users repository.UserRepository, rooms repository.RoomRepository, members repository.RoomMemberRepository, messages repository.MessageRepository, reads repository.MessageReadRepository, online UserOnlineLookup) *RoomUseCase {
	return &RoomUseCase{
		Tx:       tx,
		Users:    users,
		Rooms:    rooms,
		Members:  members,
		Messages: messages,
		Reads:    reads,
		Online:   online,
	}
}

// JoinedRoomSummary はルーム一覧向けの付加情報付き行。
type JoinedRoomSummary struct {
	Room                  entity.Room
	LastMessage           *entity.Message
	LastPreview           *string
	LastMessageSenderName *string
	UnreadCount           int64
	PeerOnline            *bool // DM の相手のみ
	DirectPeerName        *string
	DirectPeerID          *uuid.UUID
}

// RoomMemberBrief はルーム詳細のメンバー表示用。
type RoomMemberBrief struct {
	UserID   uuid.UUID
	Name     string
	JoinedAt time.Time
	Online   bool
}

// RoomDetail はルーム詳細 API 向け DTO。
type RoomDetail struct {
	Room                entity.Room
	Members             []RoomMemberBrief
	DirectPeer          *RoomMemberBrief
	MyLastReadMessageID *uuid.UUID
	UnreadCount         int64
}

// GetJoinedRooms はユーザーが参加中のルーム一覧を返す。
func (useCase *RoomUseCase) GetJoinedRooms(ctx context.Context, userID uuid.UUID) ([]entity.Room, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	return useCase.Rooms.ListJoinedRooms(ctx, userID, DefaultJoinedRoomsLimit, 0, true)
}

// IsActiveRoomMember は当該ユーザーがルームのアクティブメンバーかを返す（補助用途）。
func (useCase *RoomUseCase) IsActiveRoomMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	if roomID == uuid.Nil {
		return false, ErrInvalidRoomID
	}
	if userID == uuid.Nil {
		return false, ErrInvalidUserID
	}
	return useCase.Members.IsActiveMember(ctx, roomID, userID)
}

// CanSubscribeWebSocket は論理削除されていないルームのアクティブメンバーかつ未凍結のとき nil。
// 凍結ルームはリアルタイム購読不可（HTTP での閲覧は ListMessages 等で可）。
func (useCase *RoomUseCase) CanSubscribeWebSocket(ctx context.Context, roomID, userID uuid.UUID) error {
	r, err := RoomForActiveMember(ctx, useCase.Rooms, roomID, userID)
	if err != nil {
		return err
	}
	if r.FrozenAt != nil {
		return ErrRoomFrozen
	}
	return nil
}

// CreateOrGetDirectRoom は1対1 DM を返す。未作成なら作成し、既存ならそれを返す。
func (useCase *RoomUseCase) CreateOrGetDirectRoom(ctx context.Context, userID, targetUserID uuid.UUID) (*entity.Room, error) {
	if userID == uuid.Nil || targetUserID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if userID == targetUserID {
		return nil, ErrSelfDirect
	}
	u, err := useCase.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	if !u.Active {
		return nil, ErrInactiveUser
	}
	peer, err := useCase.Users.FindByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if peer == nil {
		return nil, ErrTargetUserNotFound
	}
	if !peer.Active {
		return nil, ErrInactiveUser
	}

	existing, err := useCase.Rooms.FindDirectRoomByUserPair(ctx, userID, targetUserID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	pairA, pairB := entity.OrderedUserPair(userID, targetUserID)

	var created *entity.Room
	err = useCase.Tx.WithinTransaction(ctx, func(ctx context.Context, rooms repository.RoomRepository, members repository.RoomMemberRepository) error {
		again, err := rooms.FindDirectRoomByUserPair(ctx, userID, targetUserID)
		if err != nil {
			return err
		}
		if again != nil {
			created = again
			return nil
		}

		now := time.Now().UTC()
		room := &entity.Room{
			ID:            uuid.New(),
			Type:          entity.RoomTypeDirect,
			Name:          nil,
			DirectUserAID: uuidPtr(pairA),
			DirectUserBID: uuidPtr(pairB),
			CreatedBy:     userID,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := rooms.Create(ctx, room); err != nil {
			return err
		}
		batch := []entity.RoomMember{
			{ID: uuid.New(), RoomID: room.ID, UserID: userID, JoinedAt: now},
			{ID: uuid.New(), RoomID: room.ID, UserID: targetUserID, JoinedAt: now},
		}
		if err := members.CreateBulk(ctx, batch); err != nil {
			return err
		}
		created = room
		return nil
	})
	if err != nil {
		if isPostgresUniqueViolation(err) {
			after, err2 := useCase.Rooms.FindDirectRoomByUserPair(ctx, userID, targetUserID)
			if err2 != nil {
				return nil, err2
			}
			if after != nil {
				return after, nil
			}
		}
		return nil, err
	}
	return created, nil
}

func (useCase *RoomUseCase) CreateGroupRoom(ctx context.Context, createdBy uuid.UUID, name string, memberIDs []uuid.UUID) (*entity.Room, error) {
	if createdBy == uuid.Nil {
		return nil, ErrInvalidCreatorID
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrGroupNameRequired
	}

	participantIDs := dedupeMemberIDs(createdBy, memberIDs)
	if len(participantIDs) < 2 {
		return nil, ErrGroupTooFewMembers
	}

	creator, err := useCase.Users.FindByID(ctx, createdBy)
	if err != nil {
		return nil, err
	}
	if creator == nil {
		return nil, ErrUserNotFound
	}
	if !creator.Active {
		return nil, ErrInactiveUser
	}

	foundUsers, err := useCase.Users.FindByIDs(ctx, participantIDs)
	if err != nil {
		return nil, err
	}
	if len(foundUsers) != len(participantIDs) {
		return nil, ErrInvalidGroupMembers
	}
	for i := range foundUsers {
		if !foundUsers[i].Active {
			return nil, ErrInactiveUser
		}
	}

	var created *entity.Room
	err = useCase.Tx.WithinTransaction(ctx, func(ctx context.Context, rooms repository.RoomRepository, members repository.RoomMemberRepository) error {
		n := name
		now := time.Now().UTC()
		room := &entity.Room{
			ID:        uuid.New(),
			Type:      entity.RoomTypeGroup,
			Name:      &n,
			CreatedBy: createdBy,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := rooms.Create(ctx, room); err != nil {
			return err
		}
		batch := make([]entity.RoomMember, 0, len(participantIDs))
		for _, uid := range participantIDs {
			batch = append(batch, entity.RoomMember{
				ID:       uuid.New(),
				RoomID:   room.ID,
				UserID:   uid,
				JoinedAt: now,
			})
		}
		if err := members.CreateBulk(ctx, batch); err != nil {
			return err
		}
		created = room
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func dedupeMemberIDs(createdBy uuid.UUID, memberIDs []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{})
	out := make([]uuid.UUID, 0, 1+len(memberIDs))
	add := func(id uuid.UUID) {
		if id == uuid.Nil {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	add(createdBy)
	for _, id := range memberIDs {
		add(id)
	}
	return out
}

func (useCase *RoomUseCase) isUserOnline(uid uuid.UUID) bool {
	if useCase.Online == nil {
		return false
	}
	return useCase.Online.IsUserOnline(uid)
}

// GetJoinedRoomSummaries は参加ルーム一覧に未読・最終メッセージ・DM 相手オンライン等を付与する。
func (useCase *RoomUseCase) GetJoinedRoomSummaries(ctx context.Context, userID uuid.UUID, limit, offset int, updatedDesc bool) ([]JoinedRoomSummary, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if limit <= 0 {
		limit = DefaultJoinedRoomsLimit
	}
	rooms, err := useCase.Rooms.ListJoinedRooms(ctx, userID, limit, offset, updatedDesc)
	if err != nil {
		return nil, err
	}
	if len(rooms) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, len(rooms))
	for i := range rooms {
		ids[i] = rooms[i].ID
	}
	lastMsgs, err := useCase.Messages.LatestPerRoom(ctx, ids)
	if err != nil {
		return nil, err
	}
	lastByRoom := make(map[uuid.UUID]entity.Message, len(lastMsgs))
	for i := range lastMsgs {
		lastByRoom[lastMsgs[i].RoomID] = lastMsgs[i]
	}
	out := make([]JoinedRoomSummary, 0, len(rooms))
	for i := range rooms {
		r := rooms[i]
		sum := JoinedRoomSummary{Room: r}
		if lm, ok := lastByRoom[r.ID]; ok {
			m := lm
			sum.LastMessage = &m
			sum.LastPreview = lastMessagePreview(&m)
			if su, err := useCase.Users.FindByID(ctx, m.SenderID); err == nil && su != nil {
				sum.LastMessageSenderName = &su.Name
			}
		}
		readRow, err := useCase.Reads.FindByRoomAndUser(ctx, r.ID, userID)
		if err != nil {
			return nil, err
		}
		var lastRead *uuid.UUID
		if readRow != nil {
			lastRead = readRow.LastReadMessageID
		}
		n, err := useCase.Messages.CountUnreadForViewer(ctx, r.ID, userID, lastRead)
		if err != nil {
			return nil, err
		}
		sum.UnreadCount = n
		if r.Type == entity.RoomTypeDirect {
			peer := directPeerUserID(&r, userID)
			if peer != nil {
				sum.DirectPeerID = peer
				u, err := useCase.Users.FindByID(ctx, *peer)
				if err != nil {
					return nil, err
				}
				if u != nil {
					sum.DirectPeerName = &u.Name
				}
				on := useCase.isUserOnline(*peer)
				sum.PeerOnline = &on
			}
		}
		out = append(out, sum)
	}
	return out, nil
}

// GetRoomDetail はルーム基本情報・メンバー・DM 相手・自分の最終既読を返す。
func (useCase *RoomUseCase) GetRoomDetail(ctx context.Context, roomID, requesterID uuid.UUID) (*RoomDetail, error) {
	if roomID == uuid.Nil || requesterID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	room, err := useCase.Rooms.FindByIDForActiveMember(ctx, roomID, requesterID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrNotRoomMember
	}
	memRows, err := useCase.Members.ListActiveMembers(ctx, roomID)
	if err != nil {
		return nil, err
	}
	userIDs := make([]uuid.UUID, 0, len(memRows))
	for _, m := range memRows {
		userIDs = append(userIDs, m.UserID)
	}
	users, err := useCase.Users.FindByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	nameByID := make(map[uuid.UUID]string, len(users))
	for _, u := range users {
		nameByID[u.ID] = u.Name
	}
	members := make([]RoomMemberBrief, 0, len(memRows))
	for _, rm := range memRows {
		members = append(members, RoomMemberBrief{
			UserID:   rm.UserID,
			Name:     nameByID[rm.UserID],
			JoinedAt: rm.JoinedAt,
			Online:   useCase.isUserOnline(rm.UserID),
		})
	}
	detail := &RoomDetail{Room: *room, Members: members}
	if room.Type == entity.RoomTypeDirect {
		if peer := directPeerUserID(room, requesterID); peer != nil {
			for i := range members {
				if members[i].UserID == *peer {
					p := members[i]
					detail.DirectPeer = &p
					break
				}
			}
		}
	}
	readRow, err := useCase.Reads.FindByRoomAndUser(ctx, roomID, requesterID)
	if err != nil {
		return nil, err
	}
	var lastRead *uuid.UUID
	if readRow != nil {
		lastRead = readRow.LastReadMessageID
		detail.MyLastReadMessageID = readRow.LastReadMessageID
	}
	unread, err := useCase.Messages.CountUnreadForViewer(ctx, roomID, requesterID, lastRead)
	if err != nil {
		return nil, err
	}
	detail.UnreadCount = unread
	return detail, nil
}

// AddGroupMember はグループにユーザーを追加する（再参加は left_at 解除）。
func (useCase *RoomUseCase) AddGroupMember(ctx context.Context, roomID, actorID, newUserID uuid.UUID) error {
	if roomID == uuid.Nil || actorID == uuid.Nil || newUserID == uuid.Nil {
		return ErrInvalidUserID
	}
	room, err := useCase.Rooms.FindByIDForActiveMember(ctx, roomID, actorID)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrNotRoomMember
	}
	if room.Type != entity.RoomTypeGroup {
		return ErrNotGroupRoom
	}
	if room.FrozenAt != nil {
		return ErrRoomFrozen
	}
	ok, err := useCase.Members.IsActiveMember(ctx, roomID, actorID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotRoomMember
	}
	u, err := useCase.Users.FindByID(ctx, newUserID)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUserNotFound
	}
	if !u.Active {
		return ErrInactiveUser
	}
	active, err := useCase.Members.IsActiveMember(ctx, roomID, newUserID)
	if err != nil {
		return err
	}
	if active {
		return ErrAlreadyMember
	}
	now := time.Now().UTC()
	prev, err := useCase.Members.FindByRoomAndUser(ctx, roomID, newUserID)
	if err != nil {
		return err
	}
	if prev != nil {
		if prev.LeftAt != nil {
			if err := useCase.Members.ReactivateMembership(ctx, roomID, newUserID, now); err != nil {
				return err
			}
			return useCase.Rooms.UpdateUpdatedAt(ctx, roomID, now)
		}
		return ErrAlreadyMember
	}
	if err := useCase.Members.Create(ctx, &entity.RoomMember{
		ID:       uuid.New(),
		RoomID:   roomID,
		UserID:   newUserID,
		JoinedAt: now,
	}); err != nil {
		return err
	}
	return useCase.Rooms.UpdateUpdatedAt(ctx, roomID, now)
}

// RemoveRoomMember は自分の退出または作成者によるキック。
func (useCase *RoomUseCase) RemoveRoomMember(ctx context.Context, roomID, actorID, targetUserID uuid.UUID) error {
	if roomID == uuid.Nil || actorID == uuid.Nil || targetUserID == uuid.Nil {
		return ErrInvalidUserID
	}
	room, err := useCase.Rooms.FindByID(ctx, roomID)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrRoomNotFound
	}
	actorMember, err := useCase.Members.IsActiveMember(ctx, roomID, actorID)
	if err != nil {
		return err
	}
	if !actorMember {
		return ErrNotRoomMember
	}
	targetOK, err := useCase.Members.IsActiveMember(ctx, roomID, targetUserID)
	if err != nil {
		return err
	}
	if !targetOK {
		return ErrMemberNotFound
	}
	if actorID != targetUserID {
		if room.CreatedBy != actorID {
			return ErrForbiddenKick
		}
	}
	now := time.Now().UTC()
	if err := useCase.Members.Leave(ctx, roomID, targetUserID, now); err != nil {
		return err
	}
	n, err := useCase.Members.CountActiveByRoomID(ctx, roomID)
	if err != nil {
		return err
	}
	if n == 0 {
		return useCase.Rooms.SoftDelete(ctx, roomID, now)
	}
	return nil
}

// UpdateGroupName はグループ名を更新する（作成者のみ、凍結中は不可）。
func (useCase *RoomUseCase) UpdateGroupName(ctx context.Context, roomID, actorID uuid.UUID, name string) error {
	if roomID == uuid.Nil || actorID == uuid.Nil {
		return ErrInvalidUserID
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrGroupNameRequired
	}
	room, err := useCase.Rooms.FindByIDForActiveMember(ctx, roomID, actorID)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrNotRoomMember
	}
	if room.Type != entity.RoomTypeGroup {
		return ErrNotGroupRoom
	}
	if room.FrozenAt != nil {
		return ErrRoomFrozen
	}
	if room.CreatedBy != actorID {
		return ErrForbiddenRoomUpdate
	}
	now := time.Now().UTC()
	return useCase.Rooms.UpdateGroupName(ctx, roomID, name, now)
}

// CloseRoom はルームを論理削除する。グループは作成者のみ、DM は参加メンバーいずれか。
func (useCase *RoomUseCase) CloseRoom(ctx context.Context, roomID, actorID uuid.UUID) error {
	if roomID == uuid.Nil || actorID == uuid.Nil {
		return ErrInvalidUserID
	}
	room, err := useCase.Rooms.FindByIDForActiveMember(ctx, roomID, actorID)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrNotRoomMember
	}
	if room.Type == entity.RoomTypeGroup && room.CreatedBy != actorID {
		return ErrForbiddenRoomUpdate
	}
	now := time.Now().UTC()
	return useCase.Rooms.SoftDelete(ctx, roomID, now)
}

func directPeerUserID(room *entity.Room, viewer uuid.UUID) *uuid.UUID {
	if room == nil || room.Type != entity.RoomTypeDirect || room.DirectUserAID == nil || room.DirectUserBID == nil {
		return nil
	}
	if *room.DirectUserAID == viewer {
		return room.DirectUserBID
	}
	if *room.DirectUserBID == viewer {
		return room.DirectUserAID
	}
	return nil
}

func lastMessagePreview(m *entity.Message) *string {
	if m == nil {
		return nil
	}
	if m.IsDeleted || m.DeletedAt != nil {
		s := "(削除されたメッセージ)"
		return &s
	}
	if m.Content != nil && *m.Content != "" {
		runes := []rune(*m.Content)
		if len(runes) > 120 {
			s := string(runes[:120]) + "…"
			return &s
		}
		return m.Content
	}
	if m.ImageURL != nil && *m.ImageURL != "" {
		s := "(画像)"
		return &s
	}
	return nil
}
