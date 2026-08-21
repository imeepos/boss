package push

// devices 内存实现(测试替身;语义与 PG 一致:同 RegistrationID 改绑)。

import (
	"context"
	"sort"
)

// DevicesMemoryStore DevicesService 内存实现。
type DevicesMemoryStore struct {
	items []Device
}

// NewDevicesMemoryStore 构造内存设备注册表。
func NewDevicesMemoryStore() *DevicesMemoryStore { return &DevicesMemoryStore{} }

// RegisterDevice 幂等注册:命中同 RegistrationID 时改绑主体。
func (s *DevicesMemoryStore) RegisterDevice(_ context.Context, subjectType string, subjectID int64, registrationID, vendor string) error {
	if !validDevice(subjectType, subjectID, registrationID) {
		return ErrInvalidDevice
	}
	for i := range s.items {
		if s.items[i].RegistrationID == registrationID {
			s.items[i].SubjectType = subjectType
			s.items[i].SubjectID = subjectID
			s.items[i].Vendor = vendor
			return nil
		}
	}
	s.items = append(s.items, Device{
		ID: int64(len(s.items) + 1), SubjectType: subjectType, SubjectID: subjectID,
		RegistrationID: registrationID, Vendor: vendor,
	})
	return nil
}

// RegistrationIDs 主体绑定的全部设备(稳定序)。
func (s *DevicesMemoryStore) RegistrationIDs(_ context.Context, subjectType string, subjectID int64) ([]string, error) {
	if subjectType != SubjectUser && subjectType != SubjectWorker || subjectID <= 0 {
		return nil, ErrInvalidDevice
	}
	out := []string{}
	for _, d := range s.items {
		if d.SubjectType == subjectType && d.SubjectID == subjectID {
			out = append(out, d.RegistrationID)
		}
	}
	sort.Strings(out)
	return out, nil
}
