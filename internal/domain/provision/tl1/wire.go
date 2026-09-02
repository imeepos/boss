package tl1

import (
	"bytes"
	"errors"
	"io"
)

// ReadFrame 逐字节读一条 TL1 报文,遇终止符解析:
//
//	; 结束报文,返回累积全文;
//	> 多块续读,继续读下一块拼接到同一报文;
//	连接关闭且无任何数据返回 io.EOF;读到存活数据后 EOF 视为报文残缺。
func ReadFrame(r io.Reader) (string, error) {
	var buf bytes.Buffer
	one := make([]byte, 1)
	for {
		if _, err := r.Read(one); err != nil {
			if errors.Is(err, io.EOF) && buf.Len() == 0 {
				return "", io.EOF
			}
			if buf.Len() > 0 {
				return "", ErrConnBroken
			}
			return "", err
		}
		b := one[0]
		switch b {
		case ';':
			return buf.String(), nil
		case '>':
			// 多块续读: 继续,不结束。
		default:
			buf.WriteByte(b)
		}
	}
}
