package lv_net

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestHttpClient(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Token")
		switch r.URL.Path {
		case "/json":
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %s", r.Header.Get("Content-Type"))
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/form":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if r.FormValue("name") != "lv" {
				t.Errorf("form name = %s", r.FormValue("name"))
			}
			_, _ = w.Write([]byte("form-ok"))
		case "/slow":
			time.Sleep(300 * time.Millisecond)
			_, _ = w.Write([]byte("slow-ok"))
		case "/err":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("denied"))
		default:
			_, _ = w.Write([]byte("get-ok"))
		}
	}))
	defer srv.Close()

	// Get 无选项（向下兼容旧调用 lv_net.Get(url)）
	if s, err := Get(srv.URL); err != nil || s != "get-ok" {
		t.Fatalf("Get = %q, %v", s, err)
	}

	// Get 单个 header
	if s, err := Get(srv.URL, WithHeader("X-Token", "abc123")); err != nil || s != "get-ok" {
		t.Fatalf("Get with header = %q, %v", s, err)
	}
	if gotHeader != "abc123" {
		t.Fatalf("server got X-Token = %q", gotHeader)
	}

	// Get 批量 header
	if s, err := Get(srv.URL, WithHeaders(map[string]string{"X-Token": "abc123", "X-Flag": "1"})); err != nil || s != "get-ok" {
		t.Fatalf("Get with headers = %q, %v", s, err)
	}

	// GetBytes
	if b, err := GetBytes(srv.URL); err != nil || string(b) != "get-ok" {
		t.Fatalf("GetBytes = %q, %v", b, err)
	}

	// PostJSON
	if b, err := PostJSON(srv.URL+"/json", map[string]any{"a": 1}, WithHeader("X-Token", "t")); err != nil || string(b) != `{"ok":true}` {
		t.Fatalf("PostJSON = %q, %v", b, err)
	}

	// PostForm
	if s, err := PostForm(srv.URL+"/form", map[string][]string{"name": {"lv"}}); err != nil || s != "form-ok" {
		t.Fatalf("PostForm = %q, %v", s, err)
	}

	// 非 2xx 应返回错误
	if _, err := Get(srv.URL + "/err"); err == nil {
		t.Fatal("expected error for status 403")
	}

	// 单请求超时不影响其它请求：慢接口用短超时应失败
	if _, err := Get(srv.URL+"/slow", WithTimeout(50*time.Millisecond)); err == nil {
		t.Fatal("expected timeout error")
	}
	// 不带超时的请求仍正常（全局默认 client 未被修改）
	if s, err := Get(srv.URL + "/slow"); err != nil || s != "slow-ok" {
		t.Fatalf("Get after timeout = %q, %v", s, err)
	}

	// GetJSON：一步解析到结构体
	var obj map[string]any
	if err := GetJSON(srv.URL+"/json", &obj); err != nil || obj["ok"] != true {
		t.Fatalf("GetJSON = %v, %v", obj, err)
	}

	// SendJSON：body 序列化 + 响应解析
	var obj2 map[string]any
	if err := SendJSON(srv.URL+"/json", map[string]int{"a": 1}, &obj2); err != nil || obj2["ok"] != true {
		t.Fatalf("SendJSON = %v, %v", obj2, err)
	}

	// SendJSON：result 传 nil 只校验状态码
	if err := SendJSON(srv.URL+"/json", map[string]int{"a": 1}, nil); err != nil {
		t.Fatalf("SendJSON nil result: %v", err)
	}

	// 非 JSON 响应应报解析错误
	if err := GetJSON(srv.URL, &obj); err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestDownload(t *testing.T) {
	chunk := bytes.Repeat([]byte("a"), 64*1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stall": // 先发一批数据，然后停发（sleep 时长即卡死时长）
			_, _ = w.Write(chunk)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(1 * time.Second)
			_, _ = w.Write(chunk)
		default: // 正常连续发送两批
			_, _ = w.Write(chunk)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(100 * time.Millisecond)
			_, _ = w.Write(chunk)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	want := int64(len(chunk) * 2)

	// 正常下载：无端到端超时，间隔 100ms 不影响结果
	path := filepath.Join(dir, "ok.bin")
	n, err := Download(srv.URL+"/file", path)
	if err != nil || n != want {
		t.Fatalf("Download = %d, %v", n, err)
	}

	// 卡死检测：第一批数据后停发，idle 200ms 应中止且清理半成品
	path = filepath.Join(dir, "stall.bin")
	if _, err := Download(srv.URL+"/stall", path, WithIdleTimeout(200*time.Millisecond)); err == nil {
		t.Fatal("expected stall error")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("partial file should be removed")
	}

	// 总预算超时：明确限制总时长也能中止
	path = filepath.Join(dir, "budget.bin")
	if _, err := Download(srv.URL+"/stall", path, WithTimeout(200*time.Millisecond)); err == nil {
		t.Fatal("expected budget timeout error")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("partial file should be removed")
	}
}

// TestDownloadLargeFile 验证大文件下载：流式写盘，内存占用不随文件大小增长
func TestDownloadLargeFile(t *testing.T) {
	const size = 50 << 20 // 50MB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(size))
		buf := make([]byte, 64*1024)
		for written := 0; written < size; {
			n := min(len(buf), size-written)
			if _, err := w.Write(buf[:n]); err != nil {
				return
			}
			written += n
		}
	}))
	defer srv.Close()

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	path := filepath.Join(t.TempDir(), "big.bin")
	n, err := Download(srv.URL, path, WithIdleTimeout(5*time.Second))
	if err != nil || n != size {
		t.Fatalf("Download = %d, %v", n, err)
	}

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	// 累计分配超过 10MB 说明把文件读进了内存（流式实现应只有常量级 buffer 分配）
	if delta := after.TotalAlloc - before.TotalAlloc; delta > 10<<20 {
		t.Fatalf("download of %dMB allocated %d bytes, not streaming", size>>20, delta)
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() != size {
		t.Fatalf("file size = %d, %v, want %d", fi.Size(), err, size)
	}
}

// TestDownloadResume 验证断点续传：Range + If-Range 校验、源变化全量重下、416 已完整、进度回调
func TestDownloadResume(t *testing.T) {
	content := bytes.Repeat([]byte("0123456789"), 10*1024)  // 100KB "旧版本"
	content2 := bytes.Repeat([]byte("abcdefghij"), 10*1024) // 100KB "新版本"
	etag := `"v1"`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If-Range 不匹配：视为源已变化，全量返回新版本
		if ir := r.Header.Get("If-Range"); ir != "" && ir != etag {
			w.Header().Set("Content-Length", strconv.Itoa(len(content2)))
			_, _ = w.Write(content2)
			return
		}
		rng := r.Header.Get("Range")
		if rng == "" {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			_, _ = w.Write(content)
			return
		}
		var start int64
		if _, err := fmt.Sscanf(rng, "bytes=%d-", &start); err != nil || start < 0 || start >= int64(len(content)) {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", len(content)))
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(content)-1, len(content)))
		w.Header().Set("Content-Length", strconv.Itoa(len(content)-int(start)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(content[start:])
	}))
	defer srv.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "file.bin")
	metaFile := path + ".lvdl"

	// 场景1：正常续传——本地已有前 40KB + 元数据，服务端补发剩余 60KB
	_ = os.WriteFile(path, content[:40*1024], 0o644)
	_ = os.WriteFile(metaFile, []byte(`{"etag":"\"v1\""}`), 0o644)
	var lastDone, lastTotal int64
	n, err := Download(srv.URL, path, WithResume(), WithProgress(func(done, total int64) {
		lastDone, lastTotal = done, total
	}))
	if err != nil || n != 60*1024 {
		t.Fatalf("resume download = %d, %v", n, err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, content) {
		t.Fatal("resumed file content mismatch")
	}
	if lastDone != int64(len(content)) || lastTotal != int64(len(content)) {
		t.Fatalf("progress = %d/%d", lastDone, lastTotal)
	}
	if _, err := os.Stat(metaFile); !os.IsNotExist(err) {
		t.Fatal("meta file should be removed after success")
	}

	// 场景2：源文件已变化（If-Range 不匹配）→ 服务器回 200 全量重下新版本
	_ = os.WriteFile(path, content[:40*1024], 0o644)
	_ = os.WriteFile(metaFile, []byte(`{"etag":"\"v9\""}`), 0o644)
	n, err = Download(srv.URL, path, WithResume())
	if err != nil || n != int64(len(content2)) {
		t.Fatalf("changed-source download = %d, %v", n, err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, content2) {
		t.Fatal("file should be fully re-downloaded new version")
	}

	// 场景3：本地已完整 → Range 超界收 416，且 Content-Range 总长吻合 → 直接成功
	_ = os.WriteFile(path, content, 0o644)
	n, err = Download(srv.URL, path, WithResume())
	if err != nil || n != 0 {
		t.Fatalf("complete download = %d, %v", n, err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, content) {
		t.Fatal("complete file should be untouched")
	}

	// 场景4：半成品无元数据（盲续）→ 不带 If-Range，服务端 206 补发
	_ = os.WriteFile(path, content[:50], 0o644)
	n, err = Download(srv.URL, path, WithResume())
	if err != nil || n != int64(len(content))-50 {
		t.Fatalf("blind resume = %d, %v", n, err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, content) {
		t.Fatal("blind resumed file content mismatch")
	}
}

// TestDownloadDefaultStallGuard 未传任何选项时也应默认开启卡死防护
func TestDownloadDefaultStallGuard(t *testing.T) {
	old := defaultIdleTimeout
	defaultIdleTimeout = 150 * time.Millisecond
	defer func() { defaultIdleTimeout = old }()

	chunk := bytes.Repeat([]byte("a"), 64*1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(chunk)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		time.Sleep(1 * time.Second) // 停流
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "stall.bin")
	if _, err := Download(srv.URL, path); err == nil {
		t.Fatal("expected default stall timeout error")
	}
}
