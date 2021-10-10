/*
 * Copyright 2021 Seth Vargo
 * Copyright 2021 Mike Helmick
 * Copyright 2021 Maxim Gulimonov
 * Copyright 2020 Ivanov Nikita
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

var _ Blob = (*FilesystemStorage)(nil)

type FilesystemStorage struct{}

func NewFilesystemStorage(_ context.Context) (Blob, error) {
	return &FilesystemStorage{}, nil
}

func (s *FilesystemStorage) CreateObject(_ context.Context, folder, filename string, contents []byte) error {
	pth := filepath.Join(folder, filename)
	if err := os.WriteFile(pth, contents, 0o600); err != nil {
		return fmt.Errorf("failed to create object: %w", err)
	}

	return nil
}

func (s *FilesystemStorage) DeleteObject(_ context.Context, folder, filename string) error {
	pth := filepath.Join(folder, filename)
	if err := os.Remove(pth); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete fs object: %w", err)
	}

	return nil
}

func (s *FilesystemStorage) GetObject(_ context.Context, folder, filename string) ([]byte, error) {
	pth := filepath.Join(folder, filename)
	b, err := os.ReadFile(pth)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("read file: %w", err)
	}

	return b, nil
}
