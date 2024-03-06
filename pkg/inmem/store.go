package inmem

import (
	"dcard-backend-2024/pkg/model"
	"fmt"
	"sort"
	"sync"
	"time"
	"unsafe"

	"github.com/biogo/store/interval"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hashicorp/go-memdb"
)

var (
	// ErrNoAdsFound is returned when the ad is not found in the store, 404
	ErrNoAdsFound error = fmt.Errorf("no ads found")
	// ErrOffsetOutOfRange is returned when the offset is out of range, 404
	ErrOffsetOutOfRange error = fmt.Errorf("offset is out of range")
	// ErrInvalidVersion is returned when the version is invalid, inconsistent with the store
	ErrInvalidVersion error = fmt.Errorf("invalid version")
)

type IntInterval struct {
	Start, End int
	UID        uintptr
	Payload    interface{} // ad id
}

func (i IntInterval) Overlap(b interval.IntRange) bool {
	return i.Start < b.End && i.End > b.Start
}

func (i IntInterval) ID() uintptr {
	return i.UID
}

func (i IntInterval) Range() interval.IntRange {
	return interval.IntRange{Start: i.Start, End: i.End}
}

type InMemoryStoreImpl struct {
	// use the Version as redis stream's message sequence number, and also store it as ad's version
	// then if the rebooted service's version is lower than the Version, it will fetch the latest ads from the db
	// and use the db's version as the Version, then start subscribing the redis stream from the Version offset
	Version          int
	ads              map[string]*model.Ad
	adsByCountry     map[string]mapset.Set[string]
	adsByGender      map[string]mapset.Set[string]
	adsByPlatform    map[string]mapset.Set[string]
	adByTimeInterval *interval.IntTree
	adByAge          *interval.IntTree
	mutex            sync.RWMutex
	memdb            *memdb.MemDB
}

func NewInMemoryStore() model.InMemoryStore {
	return &InMemoryStoreImpl{
		ads:              make(map[string]*model.Ad),
		adsByCountry:     make(map[string]mapset.Set[string]),
		adsByGender:      make(map[string]mapset.Set[string]),
		adsByPlatform:    make(map[string]mapset.Set[string]),
		adByTimeInterval: &interval.IntTree{},
		adByAge:          &interval.IntTree{},
		mutex:            sync.RWMutex{},
	}
}

// CreateBatchAds creates a batch of ads in the store
// this function does not check the version continuity.
// because if we want to support update operation restore from the snapshot,
// the version must not be continuous
// (only used in the snapshot restore)
func (s *InMemoryStoreImpl) CreateBatchAds(ads []*model.Ad) (version int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// sort the ads by version
	sort.Slice(ads, func(i, j int) bool {
		return ads[i].Version < ads[j].Version
	})

	for _, ad := range ads {
		// Update the version
		s.Version = max(s.Version, ad.Version)

		s.ads[ad.ID.String()] = ad

		// Update indexes
		for _, country := range ad.Country {
			if s.adsByCountry[country] == nil {
				s.adsByCountry[country] = mapset.NewSet[string]()
			}
			s.adsByCountry[country].Add(ad.ID.String())
		}
		for _, gender := range ad.Gender {
			if s.adsByGender[gender] == nil {
				s.adsByGender[gender] = mapset.NewSet[string]()
			}
			s.adsByGender[gender].Add(ad.ID.String())
		}
		for _, platform := range ad.Platform {
			if s.adsByPlatform[platform] == nil {
				s.adsByPlatform[platform] = mapset.NewSet[string]()
			}
			s.adsByPlatform[platform].Add(ad.ID.String())
		}
	}
	return s.Version, nil
}

func (s *InMemoryStoreImpl) CreateAd(ad *model.Ad) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if ad.Version != s.Version+1 {
		return "", ErrInvalidVersion
	}

	// Update the version
	s.Version = ad.Version

	s.ads[ad.ID.String()] = ad

	// Update indexes
	for _, country := range ad.Country {
		if s.adsByCountry[country] == nil {
			s.adsByCountry[country] = mapset.NewSet[string]()
		}
		s.adsByCountry[country].Add(ad.ID.String())
	}
	for _, gender := range ad.Gender {
		if s.adsByGender[gender] == nil {
			s.adsByGender[gender] = mapset.NewSet[string]()
		}
		s.adsByGender[gender].Add(ad.ID.String())
	}
	for _, platform := range ad.Platform {
		if s.adsByPlatform[platform] == nil {
			s.adsByPlatform[platform] = mapset.NewSet[string]()
		}
		s.adsByPlatform[platform].Add(ad.ID.String())
	}
	err := s.adByTimeInterval.Insert(
		&IntInterval{
			Start:   int(ad.StartAt.T().Unix()),
			End:     int(ad.EndAt.T().Unix()),
			UID:     uintptr(unsafe.Pointer(ad)),
			Payload: ad.ID.String(),
		}, false)

	if err != nil {
		return "", err
	}

	err = s.adByAge.Insert(
		&IntInterval{
			Start:   ad.AgeStart,
			End:     ad.AgeEnd,
			UID:     uintptr(unsafe.Pointer(ad)),
			Payload: ad.ID.String(),
		}, false)

	if err != nil {
		return "", err
	}

	return ad.ID.String(), nil
}

func (s *InMemoryStoreImpl) GetAds(req *model.GetAdRequest) (ads []*model.Ad, count int, err error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	now := time.Now()
	// nowUnix := int(now.Unix())

	// Calculate the set based on filters
	var candidateIDs mapset.Set[string]
	timeIntervalIDs := mapset.NewSet[string]()
	ageIntervalIDs := mapset.NewSet[string]()

	// filter time with the interval tree
	// timeIntervals := s.adByTimeInterval.Get(IntInterval{
	// 	Start: int(now.Unix()),
	// 	End:   int(now.Unix()),
	// })

	// for _, timeInterval := range timeIntervals {
	// 	adId := timeInterval.(*IntInterval).Payload.(string)
	// 	timeIntervalIDs.Add(adId)
	// }

	// filter age with the interval tree
	// ageIntervals := s.adByAge.Get(IntInterval{
	// 	Start: req.Age,
	// 	End:   req.Age,
	// })
	// for _, ageInterval := range ageIntervals {
	// 	adId := ageInterval.(*IntInterval).Payload.(string)
	// 	ageIntervalIDs.Add(adId)
	// }

	// intersect the time and age interval results
	if timeIntervalIDs.Cardinality() > 0 && ageIntervalIDs.Cardinality() > 0 {
		candidateIDs = timeIntervalIDs.Intersect(ageIntervalIDs)
	} else if timeIntervalIDs.Cardinality() > 0 {
		candidateIDs = timeIntervalIDs
	} else if ageIntervalIDs.Cardinality() > 0 {
		candidateIDs = ageIntervalIDs
	}

	if req.Country != "" {
		candidateIDs = s.adsByCountry[req.Country]
	}
	if req.Gender != "" {
		if candidateIDs == nil {
			candidateIDs = s.adsByGender[req.Gender]
		} else {
			candidateIDs = candidateIDs.Intersect(s.adsByGender[req.Gender])
		}
	}
	if req.Platform != "" {
		if candidateIDs == nil {
			candidateIDs = s.adsByPlatform[req.Platform]
		} else {
			candidateIDs = candidateIDs.Intersect(s.adsByPlatform[req.Platform])
		}
	}

	// If no filters are applied, use all ads
	if candidateIDs == nil {
		candidateIDs = mapset.NewSet[string]()
		for id := range s.ads {
			candidateIDs.Add(id)
		}
	}

	// Filter by time and age, and apply pagination
	for _, id := range candidateIDs.ToSlice() {
		ad := s.ads[id]
		if ad.StartAt.T().Before(now) && ad.EndAt.T().After(now) && ad.AgeStart <= req.Age && req.Age <= ad.AgeEnd {
			ads = append(ads, ad)
		}
	}

	total := len(ads)
	if total == 0 {
		return nil, 0, ErrNoAdsFound
	}

	// Apply pagination
	start := req.Offset
	if start < 0 || start >= total {
		return nil, 0, ErrOffsetOutOfRange
	}

	end := start + req.Limit
	if end > total {
		end = total
	}

	return ads[start:end], total, nil
}
