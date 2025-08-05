package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"slices"

	api "github.com/NotBalds/yacen-server/yacen_api.v2_2"
	"github.com/charmbracelet/log"
	"github.com/lib/pq"
	"google.golang.org/grpc/peer"
	"gorm.io/gorm"
)

type Room struct {
	gorm.Model
	RID string
	// PrivateInfo
	AdminKeys           pq.StringArray `gorm:"type:text[]"`
	AllowedKeys         pq.StringArray `gorm:"type:text[]"`
	PendingJoinRequests pq.StringArray `gorm:"type:text[]"`
	// PublicInfo
	Type          api.RoomType
	EncryptedName []byte `gorm:"type:bytea"`
	EncryptedDesc []byte `gorm:"type:bytea"`
}

func (s *server) CreateRoom(ctx context.Context, req *api.CreateRoomReq) (*api.CreateRoomRes, error) {
	owner, ok := CheckMeta(ctx, req)
	if !ok {
		return nil, errors.New("Wrong metadata")
	}

	err := s.AttachMeta(&ctx, req)
	if err != nil {
		log.Fatal("Could not attach meta", "err", err)
	}

	sqlDB, _ := s.DB.DB()

	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Database ping failed:", "err", err)
	}

	s.DB.AutoMigrate(&Room{})

	brid := make([]byte, 32)
	rand.Read(brid)
	rid := hex.EncodeToString(brid)

	var rooms []Room
	s.DB.Where("r_id = ?", rid).Find(&rooms)

	for len(rooms) != 0 {
		brid = make([]byte, 32)
		rand.Read(brid)
		rid = hex.EncodeToString(brid)

		var rooms []Room
		s.DB.Where("r_id = ?", rid).Find(&rooms)
	}

	s.DB.Create(&Room{
		RID:                 rid,
		AdminKeys:           []string{owner},
		AllowedKeys:         []string{owner},
		PendingJoinRequests: []string{},
		Type:                req.PublicInfo.RoomType,
		EncryptedName:       req.PublicInfo.EncryptedRoomName,
		EncryptedDesc:       req.PublicInfo.EncryptedRoomDescription,
	})

	/* rooms := make([]Room, 0)
	s.DB.Find(&rooms)

	for i := range rooms {
		log.Info("room", "id", rooms[i].ID, "type", rooms[i].Type, "admins", rooms[i].AdminKeys)
		nm := rooms[i].EncryptedName
		os.WriteFile("room_nm", nm, os.ModePerm)
	} */

	return &api.CreateRoomRes{
		RoomId: rid,
		Muid:   CreateMuid(),
	}, nil
}

func (s *server) DeleteRoom(ctx context.Context, req *api.DeleteRoomReq) (*api.DeleteRoomRes, error) {
	pkey, ok := CheckMeta(ctx, req)
	if !ok {
		return nil, errors.New("Wrong metadata")
	}

	err := s.AttachMeta(&ctx, req)
	if err != nil {
		log.Fatal("Could not attach meta", "err", err)
	}

	sqlDB, _ := s.DB.DB()

	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Database ping failed:", "err", err)
	}

	s.DB.AutoMigrate(&Room{})

	rid := req.RoomId
	var rooms []Room
	s.DB.Where("r_id = ?", rid).Find(&rooms)

	if len(rooms) == 0 {
		log.Warn("Client tried to delete room that does not exist", "rid", rid)
		return nil, errors.New("nosuchroom")
	}

	room := rooms[0]

	if !slices.Contains(room.AdminKeys, pkey) {
		p, _ := peer.FromContext(ctx)
		log.Warn("Not admin of room tried to delete it", "rid", rid, "IP", p.Addr.String(), "pubkey", pkey)
		return nil, errors.New("accessdenied")
	}

	s.DB.Where("r_id = ?", rid).Delete(&rooms)

	return &api.DeleteRoomRes{
		Muid: CreateMuid(),
	}, nil
}
