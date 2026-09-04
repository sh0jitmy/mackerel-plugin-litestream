// Copyright 2026 [Copyright Holder]
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Author: [YOUR_NAME]

package service

import (
	"context"
	"errors"

	"github.com/shjtmy/go_sh0jitmy_template/ent"
	"github.com/shjtmy/go_sh0jitmy_template/ent/user"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrUserNotFound         = errors.New("user not found")
)

type AuthService struct {
	db *ent.Client
}

// NewAuthService は AuthService のインスタンスを返します。
func NewAuthService(db *ent.Client) *AuthService {
	return &AuthService{db: db}
}

// Authenticate はユーザー名とパスワードを検証し、成功した場合はトークンを返します。
func (s *AuthService) Authenticate(ctx context.Context, username string, password string) (string, error) {
	u, err := s.db.User.Query().
		Where(user.Username(username)).
		Only(ctx)
	if err != nil {
		return "", ErrAuthenticationFailed
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", ErrAuthenticationFailed
	}

	return "secret-bearer-token", nil
}

// GetUserByUsername は指定されたユーザー名のユーザー情報を取得します。
func (s *AuthService) GetUserByUsername(ctx context.Context, username string) (*ent.User, error) {
	u, err := s.db.User.Query().
		Where(user.Username(username)).
		Only(ctx)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}
