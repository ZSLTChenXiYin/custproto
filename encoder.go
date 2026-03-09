package custproto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

// Encoder 编码器
type Encoder struct {
	writer io.Writer
	buffer *bytes.Buffer
	order  binary.ByteOrder
}

// NewEncoder 创建新的编码器
func NewEncoder(w io.Writer, order binary.ByteOrder) *Encoder {
	return &Encoder{
		writer: w,
		order:  order,
	}
}

// Encode 将结构体编码为二进制数据
func (e *Encoder) Encode(v any) error {
	e.buffer = bytes.NewBuffer(nil)

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("encoder: v must be struct or pointer to struct")
	}

	if err := e.encodeStruct(val); err != nil {
		return err
	}

	if e.writer != nil {
		_, err := e.writer.Write(e.buffer.Bytes())
		return err
	}

	return nil
}

// Bytes 返回编码后的字节数据
func (e *Encoder) Bytes() []byte {
	if e.buffer == nil {
		return nil
	}
	return e.buffer.Bytes()
}

// encodeStruct 递归编码结构体（支持内嵌结构体）
func (e *Encoder) encodeStruct(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 检查是否是内嵌结构体（匿名嵌入）
		if fieldType.Anonymous {
			// 递归编码内嵌结构体
			if err := e.encodeStruct(field); err != nil {
				return fmt.Errorf("encode embedded struct %s failed: %v", fieldType.Name, err)
			}
			continue
		}

		tag := fieldType.Tag.Get("custproto")
		if tag != "" {
			tagInfo, err := parseTag(tag)
			if err != nil {
				return err
			}

			// 如果有字段名，这是一个变长字段，直接写入数据
			if tagInfo.FieldName != "" {
				if err := e.writeField(field); err != nil {
					return err
				}
				continue
			}
		}

		// 写入普通字段
		if err := e.writeField(field); err != nil {
			return err
		}
	}

	return nil
}

// writeField 写入字段值
func (e *Encoder) writeField(field reflect.Value) error {
	switch field.Kind() {
	case reflect.Uint8:
		return binary.Write(e.buffer, e.order, uint8(field.Uint()))

	case reflect.Uint16:
		return binary.Write(e.buffer, e.order, uint16(field.Uint()))

	case reflect.Uint32:
		return binary.Write(e.buffer, e.order, uint32(field.Uint()))

	case reflect.Uint64:
		return binary.Write(e.buffer, e.order, field.Uint())

	case reflect.Int8:
		return binary.Write(e.buffer, e.order, int8(field.Int()))

	case reflect.Int16:
		return binary.Write(e.buffer, e.order, int16(field.Int()))

	case reflect.Int32:
		return binary.Write(e.buffer, e.order, int32(field.Int()))

	case reflect.Int64:
		return binary.Write(e.buffer, e.order, field.Int())

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.Uint8 {
			_, err := e.buffer.Write(field.Bytes())
			return err
		}
		return fmt.Errorf("unsupported slice type: %v", field.Type())

	default:
		return fmt.Errorf("unsupported field type: %v", field.Kind())
	}
}
