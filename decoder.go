package custproto

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

// StreamDecoder 流式解码器
type StreamDecoder struct {
	reader io.Reader
	order  binary.ByteOrder
}

// NewStreamDecoder 创建新的流式解码器
func NewStreamDecoder(r io.Reader, order binary.ByteOrder) *StreamDecoder {
	return &StreamDecoder{
		reader: r,
		order:  order,
	}
}

// Decode 从流中解码数据到结构体
func (d *StreamDecoder) Decode(v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("decoder: v must be a non-nil pointer")
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("decoder: v must point to struct")
	}

	// 创建字段值映射，用于引用前面的字段
	fieldMap := make(map[string]reflect.Value)

	return d.decodeStruct(val, fieldMap)
}

// decodeStruct 递归解码结构体
func (d *StreamDecoder) decodeStruct(val reflect.Value, fieldMap map[string]reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 将字段名和值存入映射，供后续字段引用
		fieldMap[fieldType.Name] = field

		// 处理标签
		tag := fieldType.Tag.Get("custproto")
		if tag != "" {
			tagInfo, err := parseTag(tag)
			if err != nil {
				return fmt.Errorf("parse tag failed for field %s: %v", fieldType.Name, err)
			}

			// 如果指定了字段名，表示这是一个可变长度字段
			if tagInfo.FieldName != "" {
				// 从之前解码的字段中获取长度
				lengthField, ok := fieldMap[tagInfo.FieldName]
				if !ok {
					return fmt.Errorf("field %s references unknown field %s",
						fieldType.Name, tagInfo.FieldName)
				}

				// 获取长度值
				var length uint64
				switch lengthField.Kind() {
				case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					length = lengthField.Uint()
				case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					length = uint64(lengthField.Int())
				default:
					return fmt.Errorf("field %s has unsupported length type: %v",
						tagInfo.FieldName, lengthField.Kind())
				}

				// 根据长度读取数据
				if err := d.readField(field, int(length), tagInfo.Required); err != nil {
					return fmt.Errorf("read field %s failed: %v", fieldType.Name, err)
				}
				continue
			}
		}

		// 普通字段，根据类型读取
		if err := d.readField(field, 0, false); err != nil {
			return fmt.Errorf("read field %s failed: %v", fieldType.Name, err)
		}
	}

	return nil
}

// readField 读取字段值
func (d *StreamDecoder) readField(field reflect.Value, fixedLen int, required bool) error {
	switch field.Kind() {
	case reflect.Uint8:
		var val uint8
		if err := binary.Read(d.reader, d.order, &val); err != nil {
			return err
		}
		field.SetUint(uint64(val))

	case reflect.Uint16:
		var val uint16
		if err := binary.Read(d.reader, d.order, &val); err != nil {
			return err
		}
		field.SetUint(uint64(val))

	case reflect.Uint32:
		var val uint32
		if err := binary.Read(d.reader, d.order, &val); err != nil {
			return err
		}
		field.SetUint(uint64(val))

	case reflect.Uint64:
		var val uint64
		if err := binary.Read(d.reader, d.order, &val); err != nil {
			return err
		}
		field.SetUint(val)

	case reflect.Int8:
		var val int8
		if err := binary.Read(d.reader, d.order, &val); err != nil {
			return err
		}
		field.SetInt(int64(val))

	case reflect.Int16:
		var val int16
		if err := binary.Read(d.reader, d.order, &val); err != nil {
			return err
		}
		field.SetInt(int64(val))

	case reflect.Int32:
		var val int32
		if err := binary.Read(d.reader, d.order, &val); err != nil {
			return err
		}
		field.SetInt(int64(val))

	case reflect.Int64:
		var val int64
		if err := binary.Read(d.reader, d.order, &val); err != nil {
			return err
		}
		field.SetInt(val)

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.Uint8 {
			// []byte 类型
			if fixedLen <= 0 {
				if required {
					return fmt.Errorf("required slice field requires fixed length")
				}
				// 非required字段，可以读取0长度
				field.SetBytes([]byte{})
				return nil
			}
			data := make([]byte, fixedLen)
			if _, err := io.ReadFull(d.reader, data); err != nil {
				return err
			}
			field.SetBytes(data)
		} else {
			return fmt.Errorf("unsupported slice type: %v", field.Type())
		}

	default:
		return fmt.Errorf("unsupported field type: %v", field.Kind())
	}

	return nil
}

// BufferDecoder 缓冲区解码器
type BufferDecoder struct {
	data   []byte
	order  binary.ByteOrder
	offset int
}

// NewBufferDecoder 创建新的缓冲区解码器
func NewBufferDecoder(data []byte, order binary.ByteOrder) *BufferDecoder {
	return &BufferDecoder{
		data:  data,
		order: order,
	}
}

// Decode 从缓冲区解码数据到结构体
func (b *BufferDecoder) Decode(v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("decoder: v must be a non-nil pointer")
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("decoder: v must point to struct")
	}

	// 创建字段值映射
	fieldMap := make(map[string]reflect.Value)
	b.offset = 0

	return b.decodeStruct(val, fieldMap)
}

// decodeStruct 递归解码结构体
func (b *BufferDecoder) decodeStruct(val reflect.Value, fieldMap map[string]reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 检查缓冲区是否足够
		if b.offset >= len(b.data) {
			return fmt.Errorf("buffer underflow at field %s", fieldType.Name)
		}

		// 存入字段映射
		fieldMap[fieldType.Name] = field

		// 处理标签
		tag := fieldType.Tag.Get("custproto")
		if tag != "" {
			tagInfo, err := parseTag(tag)
			if err != nil {
				return fmt.Errorf("parse tag failed for field %s: %v", fieldType.Name, err)
			}

			// 如果指定了字段名，表示可变长度字段
			if tagInfo.FieldName != "" {
				// 获取长度
				lengthField, ok := fieldMap[tagInfo.FieldName]
				if !ok {
					return fmt.Errorf("field %s references unknown field %s",
						fieldType.Name, tagInfo.FieldName)
				}

				var length uint64
				switch lengthField.Kind() {
				case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					length = lengthField.Uint()
				case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					length = uint64(lengthField.Int())
				default:
					return fmt.Errorf("field %s has unsupported length type: %v",
						tagInfo.FieldName, lengthField.Kind())
				}

				// 读取变长数据
				if err := b.readField(field, int(length), tagInfo.Required); err != nil {
					return fmt.Errorf("read field %s failed: %v", fieldType.Name, err)
				}
				continue
			}
		}

		// 普通字段
		if err := b.readField(field, 0, false); err != nil {
			return fmt.Errorf("read field %s failed: %v", fieldType.Name, err)
		}
	}

	return nil
}

// readField 读取字段值
func (b *BufferDecoder) readField(field reflect.Value, fixedLen int, required bool) error {
	switch field.Kind() {
	case reflect.Uint8:
		if b.offset+1 > len(b.data) {
			return io.EOF
		}
		field.SetUint(uint64(b.data[b.offset]))
		b.offset++

	case reflect.Uint16:
		if b.offset+2 > len(b.data) {
			return io.EOF
		}
		val := b.order.Uint16(b.data[b.offset:])
		field.SetUint(uint64(val))
		b.offset += 2

	case reflect.Uint32:
		if b.offset+4 > len(b.data) {
			return io.EOF
		}
		val := b.order.Uint32(b.data[b.offset:])
		field.SetUint(uint64(val))
		b.offset += 4

	case reflect.Uint64:
		if b.offset+8 > len(b.data) {
			return io.EOF
		}
		val := b.order.Uint64(b.data[b.offset:])
		field.SetUint(val)
		b.offset += 8

	case reflect.Int8:
		if b.offset+1 > len(b.data) {
			return io.EOF
		}
		field.SetInt(int64(int8(b.data[b.offset])))
		b.offset++

	case reflect.Int16:
		if b.offset+2 > len(b.data) {
			return io.EOF
		}
		val := int16(b.order.Uint16(b.data[b.offset:]))
		field.SetInt(int64(val))
		b.offset += 2

	case reflect.Int32:
		if b.offset+4 > len(b.data) {
			return io.EOF
		}
		val := int32(b.order.Uint32(b.data[b.offset:]))
		field.SetInt(int64(val))
		b.offset += 4

	case reflect.Int64:
		if b.offset+8 > len(b.data) {
			return io.EOF
		}
		val := int64(b.order.Uint64(b.data[b.offset:]))
		field.SetInt(val)
		b.offset += 8

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.Uint8 {
			if fixedLen <= 0 {
				if required {
					return fmt.Errorf("required slice field requires fixed length")
				}
				// 非required字段，可以设置空切片
				field.SetBytes([]byte{})
				return nil
			}
			if b.offset+fixedLen > len(b.data) {
				return io.EOF
			}
			data := make([]byte, fixedLen)
			copy(data, b.data[b.offset:b.offset+fixedLen])
			field.SetBytes(data)
			b.offset += fixedLen
		} else {
			return fmt.Errorf("unsupported slice type: %v", field.Type())
		}

	default:
		return fmt.Errorf("unsupported field type: %v", field.Kind())
	}

	return nil
}

// BytesRead 返回已读取的字节数
func (b *BufferDecoder) BytesRead() int {
	return b.offset
}

// Remaining 返回剩余未读取的字节数
func (b *BufferDecoder) Remaining() int {
	return len(b.data) - b.offset
}
