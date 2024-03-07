package greenplum

import (
	"fmt"

	_ "github.com/lib/pq"

	"github.com/greenplum-db/gp-common-go-libs/dbconn"
	"github.com/greenplum-db/gpdb/gp/constants"
	"github.com/greenplum-db/gpdb/gp/idl"
)

type Segment struct {
	Dbid          int
	Content       int
	Role          string
	PreferredRole string
	Mode          string
	Status        string
	Port          int
	Hostname      string
	Address       string
	DataDir       string
}

// IsQueryDispatcher indicates whether the segment is a QD i.e. coordinator or standby
func (seg *Segment) IsQueryDispatcher() bool {
	return seg.Content < 0
}

// IsQueryExecutor indicates whether the segment is a QE i.e. primary or mirror
func (seg *Segment) IsQueryExecutor() bool {
	return seg.Content >= 0
}

// IsActingCoordinator indicates whether the segment is currently performing the role of coordinator
func (seg *Segment) IsActingCoordinator() bool {
	return seg.IsQueryDispatcher() && seg.Role == constants.RolePrimary
}

// IsActingStandby indicates whether the segment is currently performing the role of standby
func (seg *Segment) IsActingStandby() bool {
	return seg.IsQueryDispatcher() && seg.Role == constants.RoleMirror
}

// IsActingPrimary indicates whether the segment is currently performing the role of primary
func (seg *Segment) IsActingPrimary() bool {
	return seg.Content >= 0 && seg.Role == constants.RolePrimary
}

// IsActingMirror indicates whether the segment is currently performing the role of mirror
func (seg *Segment) IsActingMirror() bool {
	return seg.Content >= 0 && seg.Role == constants.RoleMirror
}

type SegmentPair struct {
	Primary *Segment
	Mirror  *Segment
}

type GpArray struct {
	Coordinator  *Segment
	Standby      *Segment
	SegmentPairs []SegmentPair
}

// NewGpArrayFromCatalog returns a new gparray object created
// using the gp_segment_configuration catalog table
func NewGpArrayFromCatalog(conn *dbconn.DBConn) (*GpArray, error) {
	errorString := fmt.Sprintf(`failed to get %s: %%w`, constants.GpSegmentConfiguration)
	query := fmt.Sprintf("SELECT dbid, content, role, preferred_role AS preferredrole, mode, status, port, hostname, address, datadir FROM pg_catalog.%s ORDER BY content, role DESC",
		constants.GpSegmentConfiguration)

	rows, err := conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf(errorString, err)
	}

	gparray := &GpArray{}
	contentMap := make(map[int][]Segment)
	for rows.Next() {
		var seg Segment
		err = rows.StructScan(&seg)
		if err != nil {
			return nil, fmt.Errorf(errorString, err)
		}

		if seg.IsActingCoordinator() {
			gparray.Coordinator = &seg
		} else if seg.IsActingStandby() {
			gparray.Standby = &seg
		} else if seg.IsQueryExecutor() {
			contentMap[seg.Content] = append(contentMap[seg.Content], seg)
		} else {
			return nil, fmt.Errorf("invalid configuration for segment with dbid %d", seg.Dbid)
		}
	}

	pairs, err := getSegmentPairsFromContentMap(contentMap)
	if err != nil {
		return nil, err
	}
	gparray.SegmentPairs = pairs

	return gparray, nil
}

func (g *GpArray) GetPrimarySegments() []Segment {
	var segs []Segment
	for _, pairs := range g.SegmentPairs {
		if pairs.Primary != nil {
			segs = append(segs, *pairs.Primary)
		}
	}

	return segs
}

func (g *GpArray) GetMirrorSegments() []Segment {
	var segs []Segment
	for _, pairs := range g.SegmentPairs {
		if pairs.Mirror != nil {
			segs = append(segs, *pairs.Mirror)
		}
	}

	return segs
}

func (g *GpArray) GetAllSegments() []Segment {
	return append(g.GetPrimarySegments(), g.GetMirrorSegments()...)
}

func (g *GpArray) GetSegmentPairForContent(content int) (*SegmentPair, error) {
	for _, pair := range g.SegmentPairs {
		if pair.Primary.Content == content {
			return &pair, nil
		}
	}

	return nil, fmt.Errorf("could not find any segments with content %d", content)
}

func (g *GpArray) HasMirrors() bool {
	return len(g.GetMirrorSegments()) > 0
}

func (g *GpArray) GetSegmentsByHost() map[string][]Segment {
	result := make(map[string][]Segment)
	for _, seg := range g.GetAllSegments() {
		result[seg.Hostname] = append(result[seg.Hostname], seg)
	}

	return result
}

func RegisterCoordinator(seg *idl.Segment, conn *dbconn.DBConn) error {
	addCoordinatorQuery := "SELECT pg_catalog.gp_add_segment(1::int2, -1::int2, 'p', 'p', 's', 'u', '%d', '%s', '%s', '%s')"
	_, err := conn.Exec(fmt.Sprintf(addCoordinatorQuery, seg.Port, seg.HostName, seg.HostAddress, seg.DataDirectory))
	if err != nil {
		return err
	}

	return nil
}

func RegisterPrimarySegments(segs []*idl.Segment, conn *dbconn.DBConn) error {
	addPrimaryQuery := "SELECT pg_catalog.gp_add_segment_primary( '%s', '%s', %d, '%s')"
	for _, seg := range segs {
		query := fmt.Sprintf(addPrimaryQuery, seg.HostName, seg.HostAddress, seg.Port, seg.DataDirectory)

		_, err := conn.Exec(query)
		if err != nil {
			return err
		}
	}

	// FIXME: gp_add_segment_primary() starts the content ID from 1,
	// so manually update the correct values for now.
	updateContentIdQuery := "SET allow_system_table_mods=true; UPDATE gp_segment_configuration SET content = content - 1 where content > 0"
	_, err := conn.Exec(updateContentIdQuery)
	if err != nil {
		return err
	}

	return nil
}

func RegisterMirrorSegments(segs []*idl.Segment, conn *dbconn.DBConn) error {
	addMirrorQuery := "SELECT pg_catalog.gp_add_segment_mirror(%d::int2, '%s', '%s', %d, '%s');"
	for _, seg := range segs {
		query := fmt.Sprintf(addMirrorQuery, seg.Contentid, seg.HostName, seg.HostAddress, seg.Port, seg.DataDirectory)

		_, err := conn.Exec(query)
		if err != nil {
			return err
		}
	}

	return nil
}

func getSegmentPairsFromContentMap(contentMap map[int][]Segment) ([]SegmentPair, error) {
	var pairs []SegmentPair
	segsPerContent := 0

	for _, segs := range contentMap {
		if segsPerContent == 0 {
			segsPerContent = len(segs)
		} else if segsPerContent != len(segs) {
			return nil, fmt.Errorf("invalid configuration, number of segments per content is not consistent")
		}
	}

	switch segsPerContent {
	case 0:
		return nil, fmt.Errorf("invalid configuration, no segments found")

	case 1:
		for content, segs := range contentMap {
			if !segs[0].IsActingPrimary() {
				return nil, fmt.Errorf("invalid configuration, no primary segment found for content %d", content)
			}

			pairs = append(pairs, SegmentPair{Primary: &segs[0]})
		}

	case 2:
		for content, segs := range contentMap {
			if segs[0].IsActingPrimary() && segs[1].IsActingMirror() {
				pairs = append(pairs, SegmentPair{Primary: &segs[0], Mirror: &segs[1]})
			} else if segs[0].IsActingMirror() && segs[1].IsActingPrimary() {
				pairs = append(pairs, SegmentPair{Primary: &segs[1], Mirror: &segs[0]})
			} else {
				return nil, fmt.Errorf("invalid configuration, not a valid segment pair for content %d", content)
			}
		}

	default:
		return nil, fmt.Errorf("invalid configuration, found more than 2 segments per content")
	}

	return pairs, nil
}
