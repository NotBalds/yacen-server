package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/charmbracelet/log"
	"gorm.io/gorm"

	api "github.com/NotBalds/yacen-server/yacen_api.v2_2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Keys struct {
	pub  []byte
	priv []byte
}

type server struct {
	api.YacenServer
	Keys Keys
	DB   *gorm.DB
}

func CreateMuid() *api.MessageUniqueID {
	nonce := make([]byte, 24)
	rand.Read(nonce)

	return &api.MessageUniqueID{
		Nonce:     nonce,
		Timestamp: timestamppb.Now(),
	}
}

func CheckMeta(ctx context.Context, req proto.Message) (string, bool) {
	body, err := proto.Marshal(req)
	if err != nil {
		log.Error("Could not serialize request body in CreateRoom")
		return "", false
	}

	sig, pkey, err := GetMeta(ctx)
	if err != nil {
		log.Error("Could not get info from metadata")
		return "", false
	}

	_ = sig
	_ = body
	// ok := ed25519.Verify(pkey, body, sig)
	// if !ok {
	// 	p, _ := peer.FromContext(ctx)
	// 	log.Warn("Signature is wrong!", "IP", p.Addr.String())
	// 	return "", false
	// }

	bkey := base64.StdEncoding.EncodeToString(pkey)

	return bkey, true
}

func GetMeta(ctx context.Context) ([]byte, ed25519.PublicKey, error) {
	meta, _ := metadata.FromIncomingContext(ctx)

	var bsig string
	if vals, exists := meta["signature"]; exists {
		bsig = vals[0]
	} else {
		log.Warn("Could not get signature from metadata")
		return nil, nil, errors.New("nosig")
	}

	sig, err := base64.StdEncoding.DecodeString(bsig)
	if err != nil {
		log.Warn("Could not decode signature")
		return nil, nil, errors.New("badsig")
	}

	var bkey string
	if vals, exists := meta["pubkey"]; exists {
		bkey = vals[0]
	} else {
		log.Warn("Could not get pubkey from metadata")
		return nil, nil, errors.New("nokey")
	}

	pkey, err := base64.StdEncoding.DecodeString(bkey)
	if err != nil {
		log.Warn("Could not decode pubkey: Not base64")
		return nil, nil, errors.New("badkey")
	}

	if len(pkey) != ed25519.PublicKeySize {
		log.Warn("Could not decode pubkey: Wrong size")
		return nil, nil, errors.New("badkey")
	}

	return sig, pkey, nil
}

func (s server) AttachMeta(ctx *context.Context, req proto.Message) error {
	body, err := proto.Marshal(req)
	if err != nil {
		log.Error("Could not serialize request body in CreateRoom")
		return err
	}

	pubkey := s.Keys.pub
	privkey := s.Keys.priv
	bkey := base64.StdEncoding.EncodeToString(pubkey)

	sig := ed25519.Sign(privkey, body)
	bsig := base64.StdEncoding.EncodeToString(sig)

	header := metadata.New(map[string]string{
		"signature": bsig,
		"pubkey":    bkey,
	})

	*ctx = metadata.NewOutgoingContext(*ctx, header)
	return nil
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func InitKeysIfNotExist() {
	if !(FileExists("yacen-pubkey") && FileExists("yacen-privkey")) {
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)
		os.WriteFile("yacen-pubkey", pub, os.ModePerm)
		os.WriteFile("yacen-privkey", priv, os.ModePerm)
	}
}

func main() {
	port := 1448

	InitKeysIfNotExist()
	pub, _ := os.ReadFile("yacen-pubkey")
	priv, _ := os.ReadFile("yacen-privkey")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	db := newDB()
	api.RegisterYacenServer(s, &server{DB: db, Keys: Keys{pub, priv}})
	reflection.Register(s)
	log.Info("server listening at", "addr", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatal("failed to serve:", "err", err)
	}
}
