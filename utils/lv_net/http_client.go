package lv_net

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// defaultTimeout 全局兜底超时，单次请求可用 WithTimeout 覆盖
const defaultTimeout = 10 * time.Second

// defaultClient 包级共享客户端，http.Client 并发安全，共享可复用连接池
var defaultClient = &http.Client{
	Transport: http.DefaultTransport,
	Timeout:   defaultTimeout,
}

// downloadClient 下载专用客户端：无端到端超时（避免误杀大文件传输），
// 仅靠传输层阶段超时兜底——拨号 30s、TLS 10s、响应头 30s
var downloadClient = &http.Client{
	Transport: func() http.RoundTripper {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.ResponseHeaderTimeout = 30 * time.Second
		return t
	}(),
}

// requestOptions 请求配置
type requestOptions struct {
	headers     map[string]string
	timeout     time.Duration
	idleTimeout time.Duration
	client      *http.Client
	resume      bool
	progress    func(done, total int64)
}

// RequestOption 请求可选项
type RequestOption func(*requestOptions)

// WithHeader 设置单个请求 header
func WithHeader(key, value string) RequestOption {
	return func(o *requestOptions) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[key] = value
	}
}

// WithHeaders 批量设置请求 header
func WithHeaders(headers map[string]string) RequestOption {
	return func(o *requestOptions) {
		if o.headers == nil {
			o.headers = make(map[string]string, len(headers))
		}
		maps.Copy(o.headers, headers)
	}
}

// WithTimeout 设置本次请求的总预算超时，通过 context 实现，不影响其它请求
func WithTimeout(d time.Duration) RequestOption {
	return func(o *requestOptions) { o.timeout = d }
}

// WithIdleTimeout 自定义下载空闲超时：每收到一批数据就重置计时，超过 d 无数据才中止。
// Download 默认已开启 60s 卡死防护，传负数可禁用
func WithIdleTimeout(d time.Duration) RequestOption {
	return func(o *requestOptions) { o.idleTimeout = d }
}

// WithResume 启用断点续传：本地存在半成品时从已下载偏移继续（Range 请求），
// 并通过 If-Range（ETag/Last-Modified）确保源文件未变化——变化时服务器回 200，自动全量重下。
// 中断时保留半成品与 "<path>.lvdl" 元数据供下次续传，完成后自动清理。
// 注意：若服务器不返回 ETag/Last-Modified，将无法校验源变化，续传有拼出坏文件的风险。
func WithResume() RequestOption {
	return func(o *requestOptions) { o.resume = true }
}

// WithProgress 设置下载进度回调：done 为已下载字节数（含续传偏移），total 为总字节数（未知时为 -1）。
// 每写入一批数据触发一次，频率较高时请自行节流
func WithProgress(fn func(done, total int64)) RequestOption {
	return func(o *requestOptions) { o.progress = fn }
}

// WithClient 使用自定义 client（代理、mTLS 等场景），不传则用全局默认
func WithClient(c *http.Client) RequestOption {
	return func(o *requestOptions) { o.client = c }
}

func newRequestOptions(opts []RequestOption) *requestOptions {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Get 发送 GET 请求，返回响应文本
func Get(rawURL string, opts ...RequestOption) (string, error) {
	data, err := doRequest(http.MethodGet, rawURL, nil, "", newRequestOptions(opts))
	return string(data), err
}

// GetBytes 发送 GET 请求，返回原始字节，适合下载文件、图片验证码等场景
func GetBytes(rawURL string, opts ...RequestOption) ([]byte, error) {
	return doRequest(http.MethodGet, rawURL, nil, "", newRequestOptions(opts))
}

// PostJSON 发送 POST 请求，body 自动序列化为 JSON，返回原始响应
func PostJSON(rawURL string, body any, opts ...RequestOption) ([]byte, error) {
	reader, err := marshalBody(body)
	if err != nil {
		return nil, err
	}
	return doRequest(http.MethodPost, rawURL, reader, "application/json", newRequestOptions(opts))
}

// GetJSON 发送 GET 请求并把 JSON 响应解析到 result（result 传 nil 则只校验状态码）
func GetJSON(rawURL string, result any, opts ...RequestOption) error {
	return doJSON(http.MethodGet, rawURL, nil, result, newRequestOptions(opts))
}

// SendJSON 发送 POST 请求：body 自动序列化为 JSON，响应解析到 result（result 传 nil 则忽略响应体）
func SendJSON(rawURL string, body, result any, opts ...RequestOption) error {
	return doJSON(http.MethodPost, rawURL, body, result, newRequestOptions(opts))
}

func doJSON(method, rawURL string, body, result any, o *requestOptions) error {
	reader, err := marshalBody(body)
	if err != nil {
		return err
	}
	data, err := doRequest(method, rawURL, reader, "application/json", o)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	if err = json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}
	return nil
}

func marshalBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("json marshal: %w", err)
	}
	return bytes.NewReader(data), nil
}

// PostForm 发送表单 POST 请求，返回响应文本
func PostForm(rawURL string, data url.Values, opts ...RequestOption) (string, error) {
	reader := strings.NewReader(data.Encode())
	res, err := doRequest(http.MethodPost, rawURL, reader, "application/x-www-form-urlencoded", newRequestOptions(opts))
	return string(res), err
}

// errRestartDownload 内部哨兵：本地半成品无效（如 416 且大小不符），需不带 Range 全量重下
var errRestartDownload = errors.New("restart download")

// defaultIdleTimeout 下载默认卡死防护：数据持续流入则不限总时长，停流超过该时长才中止
var defaultIdleTimeout = 60 * time.Second

// Download 流式下载文件到本地，适合大文件：
//   - 默认无端到端超时（不怕大文件慢），默认 60s 卡死防护（停流即中止）
//   - io.Copy 流式写盘，内存占用恒定
//   - WithTimeout(d)：限制总时长预算；WithIdleTimeout(d)：自定义/禁用（负数）卡死检测
//   - WithResume()：断点续传（Range + If-Range 校验），中断时保留半成品与 "<path>.lvdl" 元数据
//   - WithProgress(fn)：下载进度回调
//   - 未启用续传时失败自动删除半成品
//
// 返回本次实际写入的字节数（续传时不包含此前已下载的部分）
func Download(rawURL, filePath string, opts ...RequestOption) (int64, error) {
	o := newRequestOptions(opts)
	client := o.client
	if client == nil {
		client = downloadClient
	}
	// 最多两趟：续传遇 416 且半成品损坏时，去掉 Range 全量重下一趟
	for attempt := 0; ; attempt++ {
		n, err := downloadOnce(client, o, rawURL, filePath, attempt > 0)
		if err == errRestartDownload && attempt == 0 {
			continue
		}
		return n, err
	}
}

func downloadOnce(client *http.Client, o *requestOptions, rawURL, filePath string, fresh bool) (int64, error) {
	ctx := context.Background()
	if o.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}
	// 卡死检测需要可取消的 context，必须在构建请求前创建，cancel 才能中断 body 读取
	var stallCancel context.CancelFunc
	idle := o.idleTimeout
	if idle == 0 {
		idle = defaultIdleTimeout // 未指定时默认开启卡死防护，负数表示禁用
	}
	if idle > 0 {
		ctx, stallCancel = context.WithCancel(ctx)
		defer stallCancel()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	for k, v := range o.headers {
		request.Header.Set(k, v)
	}

	// 断点续传：本地有半成品则从偏移处继续，If-Range 让服务器校验源文件未变化
	metaFile := filePath + ".lvdl"
	var offset int64
	if o.resume && !fresh {
		if fi, err := os.Stat(filePath); err == nil && fi.Size() > 0 {
			offset = fi.Size()
			request.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
			if meta, err := readMeta(metaFile); err == nil {
				if meta.ETag != "" {
					request.Header.Set("If-Range", meta.ETag)
				} else if meta.LastModified != "" {
					request.Header.Set("If-Range", meta.LastModified)
				}
			}
		}
	}

	res, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	appendMode := false
	switch res.StatusCode {
	case http.StatusPartialContent: // 206：服务器确认从未下载处继续
		appendMode = true
	case http.StatusRequestedRangeNotSatisfiable: // 416
		if total, ok := parseContentRangeTotal(res.Header.Get("Content-Range")); ok && offset == total {
			_ = os.Remove(metaFile)
			return 0, nil // 本地已下载完整，直接成功
		}
		return 0, errRestartDownload // 半成品损坏，全量重下
	default:
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return 0, fmt.Errorf("http status %d", res.StatusCode)
		}
	}

	var f *os.File
	if appendMode {
		f, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	} else {
		f, err = os.Create(filePath) // 200：全量重下（服务器不支持 Range 或源已变化）
	}
	if err != nil {
		return 0, err
	}

	if o.resume {
		saveMeta(metaFile, res.Header.Get("ETag"), res.Header.Get("Last-Modified"))
	}

	total := offset + res.ContentLength // Content-Length 未知时为 -1
	var w io.Writer = f
	if o.progress != nil {
		w = &progressWriter{w: f, done: offset, total: total, onProg: o.progress}
	}

	// 卡死检测：包一层 reader，每次读到数据就重置计时器，超时触发 cancel 让阻塞中的 Read 立刻报错
	var body io.Reader = res.Body
	stopStall := func() {}
	if stallCancel != nil {
		sr := newStallReader(res.Body, idle, stallCancel)
		body = sr
		stopStall = sr.stop
	}

	n, copyErr := io.Copy(w, body)
	stopStall()
	closeErr := f.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		if !o.resume {
			_ = os.Remove(filePath) // 未启用续传：清理半成品
		}
		return n, copyErr
	}
	_ = os.Remove(metaFile) // 下载完成，清理续传元数据
	return n, nil
}

// downloadMeta 续传元数据：记录下载开始时的文件校验信息
type downloadMeta struct {
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
}

func saveMeta(path, etag, lastModified string) {
	if etag == "" && lastModified == "" {
		return
	}
	if data, err := json.Marshal(downloadMeta{ETag: etag, LastModified: lastModified}); err == nil {
		_ = os.WriteFile(path, data, 0o644)
	}
}

func readMeta(path string) (downloadMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return downloadMeta{}, err
	}
	var m downloadMeta
	err = json.Unmarshal(data, &m)
	return m, err
}

// parseContentRangeTotal 从 Content-Range 头取总大小，支持 "bytes 100-200/12345" 与 "bytes */12345"
func parseContentRangeTotal(cr string) (int64, bool) {
	idx := strings.LastIndex(cr, "/")
	if idx < 0 || idx == len(cr)-1 {
		return 0, false
	}
	total, err := strconv.ParseInt(cr[idx+1:], 10, 64)
	if err != nil || total < 0 {
		return 0, false
	}
	return total, true
}

// progressWriter 进度回调 writer
type progressWriter struct {
	w      io.Writer
	done   int64
	total  int64
	onProg func(done, total int64)
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	if n > 0 && p.onProg != nil {
		p.done += int64(n)
		p.onProg(p.done, p.total)
	}
	return n, err
}

// stallReader 卡死检测 reader：每次 Read 前重置定时器，定时器到期触发 cancel
type stallReader struct {
	r      io.Reader
	d      time.Duration
	timer  *time.Timer
	cancel context.CancelFunc
}

func newStallReader(r io.Reader, d time.Duration, cancel context.CancelFunc) *stallReader {
	return &stallReader{
		r:      r,
		d:      d,
		timer:  time.AfterFunc(d, cancel),
		cancel: cancel,
	}
}

func (s *stallReader) Read(p []byte) (int, error) {
	s.timer.Reset(s.d)
	n, err := s.r.Read(p)
	return n, err
}

func (s *stallReader) stop() {
	s.timer.Stop()
}

// doRequest 统一请求入口：构建请求、设置 header、检查状态码、读取响应
func doRequest(method, rawURL string, body io.Reader, contentType string, o *requestOptions) ([]byte, error) {
	client := o.client
	if client == nil {
		client = defaultClient
	}

	ctx := context.Background()
	if o.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}

	request, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	for k, v := range o.headers {
		request.Header.Set(k, v)
	}

	res, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return resBody, fmt.Errorf("http status %d, body: %.200s", res.StatusCode, resBody)
	}
	return resBody, nil
}
