package tl1

import (
	"context"
	"strconv"
	"strings"

	"github.com/ymm-001/boss/internal/domain/provision"
)

// traceEntry 单条命令交互留痕:请求行 + 应答(设备原始报文或错误文本)。
type traceEntry struct {
	cmd     string
	resp    string
	errResp bool // resp 为错误文本而非设备应答(断线/构建失败)
	attempt int  // 尝试序号,1 起;断线整段重跑时递增
}

// traceSink 记录型命令口:包装真实 sink,逐条留指令与应答原始报文,
// 按尝试分组缓存;收口(finalize)时再决定取舍,被重连重试替代的
// 连接错误不混入最终成功证据,只在失败时作为上下文显式保留。
type traceSink struct {
	inner   CmdSink
	trace   *provision.ExecTrace
	seq     int
	attempt int
	entries []traceEntry
}

// newTraceSink 构造记录口,尝试序号从 1 起。
func newTraceSink(t *provision.ExecTrace) *traceSink {
	return &traceSink{trace: t, attempt: 1}
}

// Do 记录一次命令交互;留痕 ctag 与线上对齐:优先取响应实际 ctag
// (业务 Tag 或会话自增),无应答(断线/构建失败)才退回 TRC<seq> 占位,不留 B 位假象。
func (s *traceSink) Do(ctx context.Context, c Command) (*Response, error) {
	s.seq++
	r, err := s.inner.Do(ctx, c)
	tag := "TRC" + strconv.Itoa(s.seq)
	if r != nil && r.CTag != "" {
		tag = r.CTag
	}
	line, _ := Build(c, tag)
	ent := traceEntry{cmd: string(line), attempt: s.attempt}
	switch {
	case r != nil && r.Raw != "":
		ent.resp = strings.TrimRight(r.Raw, "\n")
	case err != nil:
		ent.resp, ent.errResp = err.Error(), true
	}
	s.entries = append(s.entries, ent)
	return r, err
}

// nextAttempt 断线重连、整段重跑前调用:先前尝试整组降级为被替代上下文。
func (s *traceSink) nextAttempt() { s.attempt++ }

// finalize 收口 trace(Exec 末尾调用一次):
//   - 成功:只留最终尝试,指令与应答逐条 1:1,ctag 可对齐,
//     被重连重试替代的连接错误不混入成功证据;
//   - 失败:全部尝试按时间序保留,被替代尝试以显式标头分隔,
//     结尾附 EXEC-ERROR,错误上下文不静默丢失。
func (s *traceSink) finalize(err error) {
	var cmds []string
	var resp string
	for a := 1; a <= s.attempt; a++ {
		group := entriesOfAttempt(s.entries, a)
		if len(group) == 0 {
			continue
		}
		if err == nil && a != s.attempt {
			continue
		}
		if a != s.attempt {
			resp = joinResp(resp, "[attempt "+strconv.Itoa(a)+" superseded by reconnect]")
		}
		for _, e := range group {
			cmds = append(cmds, e.cmd)
			if e.resp != "" {
				resp = joinResp(resp, e.resp)
			}
		}
	}
	if err != nil {
		resp = joinResp(resp, "EXEC-ERROR: "+err.Error())
	}
	if cmds == nil {
		cmds = []string{}
	}
	s.trace.Commands = cmds
	s.trace.Response = resp
}

// entriesOfAttempt 取指定尝试的留痕子序列(保持原序)。
func entriesOfAttempt(entries []traceEntry, attempt int) []traceEntry {
	var out []traceEntry
	for _, e := range entries {
		if e.attempt == attempt {
			out = append(out, e)
		}
	}
	return out
}

// joinResp 追加一段应答,分隔行隔开多次交互;
// 分隔符固定 4 横线,与设备表格分隔行(5 横线)区分,可安全再切分。
func joinResp(prev, add string) string {
	if prev == "" {
		return add
	}
	return prev + "\n----\n" + add
}
