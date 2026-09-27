package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Protocol struct {
	reader *bufio.Reader
	writer io.Writer
}

func NewProtocol(in io.Reader, out io.Writer) *Protocol {
	return &Protocol{
		reader: bufio.NewReader(in),
		writer: out,
	}
}

func (p *Protocol) ReadMessage() (map[string]any, error) {
	headers := map[string]string{}
	for {
		line, err := p.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil, io.EOF
			}
			return nil, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header: %s", line)
		}
		headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
	}

	lengthValue, ok := headers["content-length"]
	if !ok {
		return nil, fmt.Errorf("missing content-length header")
	}
	length, err := strconv.Atoi(lengthValue)
	if err != nil || length <= 0 {
		return nil, fmt.Errorf("invalid content-length: %s", lengthValue)
	}

	body := make([]byte, length)
	if _, err = io.ReadFull(p.reader, body); err != nil {
		return nil, err
	}

	var message map[string]any
	if err = json.Unmarshal(body, &message); err != nil {
		return nil, err
	}
	return message, nil
}

func (p *Protocol) WriteMessage(message map[string]any) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body)))
	buf.Write(body)
	_, err = p.writer.Write(buf.Bytes())
	return err
}
