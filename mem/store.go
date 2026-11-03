package mem

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

type Store struct {
	dir   string
	cache map[string]*Session
}

type sessionHeader struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func NewStore(dir string) (*Store, error) {
	// 创建目录，支持已存在目录
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &Store{
		dir:   dir,
		cache: make(map[string]*Session),
	}, nil
}

// createSession 创建一个 Session
func createSession(id, dir string) (*Session, error) {
	// 校验ID是否合法
	if err := validateID(id); err != nil {
		return nil, err
	}

	// 构造 header
	now := time.Now().UTC()
	header := &sessionHeader{
		Type:      "session",
		ID:        id,
		CreatedAt: now,
	}

	// 序列化 header
	data, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')

	// 构造 filePath
	path := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, err
	}

	return &Session{
		ID:        id,
		CreatedAt: now,
		msgs:      make([]*schema.AgenticMessage, 0),
		filePath:  path,
	}, nil
}

// loadSession 从 filePath 加载 Session
func loadSession(filePath string) (*Session, error) {
	// 打开文件
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	// 由于文件的第一行需要保存 Session 的信息
	// 因此读取第一行时先提取 sessionHeader
	// 以确认是哪一个 Session
	isHeader := true

	session := &Session{
		filePath: filePath,
	}
	msgs := make([]*schema.AgenticMessage, 0)

	// 逐行扫描
	for scanner.Scan() {
		line := scanner.Bytes()
		if isHeader {
			header := &sessionHeader{}
			// 反序列化
			if err := json.Unmarshal(line, header); err != nil {
				return nil, err
			}
			// 未知 header 类型
			if header.Type != "session" {
				return nil, fmt.Errorf("invalid session header: %s", string(line))
			}

			session.CreatedAt = header.CreatedAt
			session.ID = header.ID

			isHeader = false
			continue
		}

		// 非 header 就是 msg
		// 逐个 append 到 msgs 中
		msg := &schema.AgenticMessage{}
		if err := json.Unmarshal(line, msg); err != nil {
			return nil, err
		}

		msgs = append(msgs, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 没有读到 header 信息，说明文件为空
	if isHeader {
		return nil, fmt.Errorf("session header not found in %s", filePath)
	}

	session.msgs = msgs
	return session, nil
}

// GetOrCreate 获取或者创建 Session
// 若缓存命中，直接返回 Session
// 若不存在，创建一个 Session 并且写入缓存
// 若存在，直接返回 Session 并保证写入缓存
func (s *Store) GetOrCreate(id string) (*Session, error) {
	// 校验 ID 是否合法
	if err := validateID(id); err != nil {
		return nil, err
	}

	if session, ok := s.cache[id]; ok {
		// 发现 header 中的 id 与需要获得的文件 id 不同
		if session.ID != id {
			return nil, fmt.Errorf("invalid session id: %s", id)
		}
		return session, nil
	}

	// 构造 filePath
	path := filepath.Join(s.dir, id+".jsonl")
	_, err := os.Stat(path)
	// 文件不存在
	if errors.Is(err, os.ErrNotExist) {
		session, err := createSession(id, s.dir)
		if err != nil {
			return nil, err
		}
		// 写缓存
		s.cache[id] = session
		return session, nil
	}
	// 其他错误
	if err != nil {
		return nil, err
	}

	// 不在缓存
	session, err := loadSession(path)
	if err != nil {
		return nil, err
	}

	// 发现 header 中的 id 与需要获得的文件 id 不同
	if session.ID != id {
		return nil, fmt.Errorf("invalid session id: %s", id)
	}

	// 保证在缓存中
	s.cache[id] = session

	return session, nil
}

// List 列出该目录下的所有 Session
func (s *Store) List() ([]*Session, error) {
	// 读出该目录下所有的文件
	fs, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	sessions := make([]*Session, 0)
	for _, f := range fs {
		// 跳过目录
		if f.IsDir() {
			continue
		}
		name := f.Name()
		// 跳过后缀非 .jsonl
		if filepath.Ext(name) != ".jsonl" {
			continue
		}
		// 在缓存中直接读取
		id := strings.TrimSuffix(name, ".jsonl")
		session, ok := s.cache[id]
		// 不在缓存中
		if !ok {
			session, err = loadSession(filepath.Join(s.dir, name))
			if err != nil {
				return nil, fmt.Errorf("invalid session %s: %w", name, err)
			}

		}
		// 读取出的文件 id 与 sessionId不同
		if session.ID != id {
			return nil, fmt.Errorf("invalid session id: %s", id)
		}
		// 插入缓存
		if !ok {
			s.cache[id] = session
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Delete 删除一个 Session 并且清除 Session
func (s *Store) Delete(id string) error {
	// 校验 ID 是否合法
	if err := validateID(id); err != nil {
		return err
	}

	// 构造 filePath
	path := filepath.Join(s.dir, id+".jsonl")

	// 删除 Session
	err := os.Remove(path)
	if err != nil {
		return err
	}

	// 从缓存中删除 Session
	delete(s.cache, id)

	return nil
}

// validateID 校验 ID 是否合法，不合法返回 error
func validateID(id string) error {
	if id == "" || filepath.Base(id) != id {
		return fmt.Errorf("invalid id %s", id)
	}
	return nil
}
