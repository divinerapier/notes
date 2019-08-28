# INTRODUCE TO SEAWEEDFS

## 概述

Seaweedfs 上传文件

### Assign

`Master Server` 提供 `grpc` 接口 `Assign`, 根据请求参数, 从一个可写`Volume`分配出一个 `File ID`

``` protobuf3
// TODO: Assign function signature

```

如果当前没有可写`Volume`时，并且`Volume` 数量不足最大值时`(ms.Topo.FreeSpace() > 0)`, 会尝试扩容

``` go
// Master Server
func (vg *VolumeGrowth) AutomaticGrowByType(option *VolumeGrowOption, grpcDialOption grpc.DialOption, topo *Topology) (count int, err error) {
    return vg.GrowByCountAndType(grpcDialOption, vg.findVolumeCount(option.ReplicaPlacement.GetCopyCount()), option, topo)
}
func (vg *VolumeGrowth) GrowByCountAndType(grpcDialOption grpc.DialOption, targetCount int, option *VolumeGrowOption, topo *Topology) (counter int, err error) {
    for i := 0; i < targetCount; i++ {
        if c, e := vg.findAndGrow(grpcDialOption, topo, option); e == nil {
            counter += c
        } else {
            return counter, e
        }
    }
    return
}
func (vg *VolumeGrowth) findAndGrow(grpcDialOption grpc.DialOption, topo *Topology, option *VolumeGrowOption) int {
    servers := vg.findEmptySlotsForOneVolume(topo, option)
    vid := topo.NextVolumeId()
    vg.grow(grpcDialOption, topo, vid, option, servers...)
    return len(servers)
}

// 目前不清楚这个函数的作用
func (vg *VolumeGrowth) findEmptySlotsForOneVolume(topo *Topology, option *VolumeGrowOption) (servers []*DataNode, err error) {
    return
}

func (vg *VolumeGrowth) grow(grpcDialOption grpc.DialOption, topo *Topology, vid storage.VolumeId, option *VolumeGrowOption, servers ...*DataNode) error {
    for _, server := range servers {
        // 调用 volume service 分配 volume
        AllocateVolume(server, grpcDialOption, vid, option)
        vi := storage.VolumeInfo{
            Id:               vid,
            Size:             0,
            Collection:       option.Collection,
            ReplicaPlacement: option.ReplicaPlacement,
            Ttl:              option.Ttl,
            Version:          storage.CurrentVersion,
        }
        server.AddOrUpdateVolume(vi)
        topo.RegisterVolumeLayout(vi, server)
    }
    return
}

// 神奇的函数，输入 ReplicaPlacement.GetCopyCount() 得到volume数量
func (vg *VolumeGrowth) findVolumeCount(copyCount int) (count int) {
    switch copyCount {
    case 1:
        count = 7
    case 2:
        count = 6
    case 3:
        count = 3
    default:
        count = 1
    }
    return
}

```

``` go
// Volume Server

// Volume Server grpc api
func (vs *VolumeServer) AllocateVolume(ctx context.Context, req *volume_server_pb.AllocateVolumeRequest) {
    vs.store.AddVolume(
        storage.VolumeId(req.VolumeId),
        req.Collection,
        vs.needleMapKind,
        req.Replication,
        req.Ttl,
        req.Preallocate,
    )
}

func (s *Store) AddVolume(volumeId VolumeId, collection string, needleMapKind NeedleMapType, replicaPlacement string, ttlString string, preallocate int64) error {
    // replicaPlacement 是一个长度为3的字符串
    // 每一位都是一个 0-9 的数字
    rt := NewReplicaPlacementFromString(replicaPlacement)
    ttl := ReadTTL(ttlString)
    return s.addVolume(volumeId, collection, needleMapKind, rt, ttl, preallocate)
}

func (s *Store) addVolume(vid VolumeId, collection string, needleMapKind NeedleMapType, replicaPlacement *ReplicaPlacement, ttl *TTL, preallocate int64) error {
    if s.findVolume(vid) != nil {
        return
    }
    // 找到 (location.MaxVolumeCount - location.VolumesLen()) 值最大的 Location
    location := s.FindFreeLocation()
    volume = NewVolume(location.Directory, collection, vid, needleMapKind, replicaPlacement, ttl, preallocate)
    location.SetVolume(vid, volume)
    s.NewVolumeIdChan <- vid
}

func (s *Store) findVolume(vid VolumeId) *Volume {
    for _, location := range s.Locations {
        if v, found := location.FindVolume(vid); found {
            return v
        }
    }
    return nil
}

func NewVolume(dirname string, collection string, id VolumeId, needleMapKind NeedleMapType, replicaPlacement *ReplicaPlacement, ttl *TTL, preallocate int64) (v *Volume, e error) {
    // if replicaPlacement is nil, the superblock will be loaded from disk
    v = &Volume{dir: dirname, Collection: collection, Id: id}
    v.SuperBlock = SuperBlock{ReplicaPlacement: replicaPlacement, Ttl: ttl}
    v.needleMapKind = needleMapKind
    e = v.load(true, true, needleMapKind, preallocate)
    return
}

func (v *Volume) load(
    alsoLoadIndex bool,
    createDatIfMissing bool,
    needleMapKind NeedleMapType,
    preallocate int64) error {

    fileName := v.FileName()
    alreadyHasSuperBlock := false

    exists, canRead, canWrite, modifiedTime, fileSize := checkFile(fileName + ".dat")
    alreadyHasSuperBlock = exists && fileSize >= _SuperBlockSize
    if !exists {
        if !createDatIfMissing {
            return
        }
        v.dataFile = createVolumeFile(fileName+".dat", preallocate)
    }

    if alreadyHasSuperBlock {
        v.readSuperBlock()
    } else {
        v.maybeWriteSuperBlock()
    }
    if alsoLoadIndex {
        return nil
    }
    var indexFile *os.File
    if v.readOnly {
        indexFile = os.OpenFile(fileName+".idx", os.O_RDONLY, 0644)
    } else {
        indexFile = os.OpenFile(fileName+".idx", os.O_RDWR|os.O_CREATE, 0644)
    }
    v.readOnly = CheckVolumeDataIntegrity(v, indexFile) != nil
    switch needleMapKind {
    case NeedleMapInMemory:
        v.nm = LoadCompactNeedleMap(indexFile)
    }
    return
}

func LoadCompactNeedleMap(file *os.File) (*NeedleMap, error) {
    nm := NewCompactNeedleMap(file)
    return doLoading(file, nm)
}

func doLoading(file *os.File, nm *NeedleMap) (*NeedleMap, error) {
    e := WalkIndexFile(file, func(key types.NeedleId, offset types.Offset, size uint32) error {
        nm.MaybeSetMaxFileKey(key)
        // 更新文件数量及大小； 如果key(needle id) 已经存在，则将原来的文件标记为已删除
        // 同时更新已删除信息
        if !offset.IsZero() && size != types.TombstoneFileSize {
            nm.FileCounter++
            nm.FileByteCounter = nm.FileByteCounter + uint64(size)
            oldOffset, oldSize := nm.m.Set(types.NeedleId(key), offset, size)
            if !oldOffset.IsZero() && oldSize != types.TombstoneFileSize {
                nm.DeletionCounter++
                nm.DeletionByteCounter = nm.DeletionByteCounter + uint64(oldSize)
            }
        } else {
            oldSize := nm.m.Delete(types.NeedleId(key))
            nm.DeletionCounter++
            nm.DeletionByteCounter = nm.DeletionByteCounter + uint64(oldSize)
        }
        return nil
    })
    return nm, e
}

```

## 数据结构

### Volume

``` go

type Volume struct {
    SuperBlock SuperBlock
    Needles    []Needle
}

```

#### SuperBlock

参考 `volume_super_block.go:ReadSuperBlock`

``` go
type SuperBlock struct {
    Header [8]byte
}

┌-------------------------------------------------------------------------------┐
| 0       | 1       | 2                 | 4                 | 6                 |
| version | rp      | ttl               | cr                | es                |
└-------------------------------------------------------------------------------┘
rp: replica placement
cr: compact revision
es: extra size
```

#### Needle

##### Index

参考 `volume_read_write.go:(*Volume).writeNeedle`, `volume_read_write.go:(*Volume).readNeedle`

以 `Version3` 为🌰

``` go
type NeedleValue struct {
    Key    NeedleId
    Offset Offset `comment:"Volume offset"` //since aligned to 8 bytes, range is 4G*8=32G
    Size   uint32 `comment:"Size of the data portion"`
}
type Offset struct {
    OffsetHigher
    OffsetLower
}

```

##### Data

``` go
R: required, O: optional
----------┬-------------------------------------------------------------------------------┐
R/O|Offset|    0    |    1    |    2    |    3    |    4    |    5    |    6    |    7    |
----------┼-------------------------------------------------------------------------------┤
R  | 0    | cookie                                | needle id part 0                      |
R  | 8    | needle id part 1                      | size                                  |
----------┼-------------------------------------------------------------------------------┤
O  |16    | data_size: len(needle.data)           | needle.data                           |
O  |      | flag    | some fields                                                         |
O  |      | current_timestamp                                                             |
O  |      | checksum                              | padding                               |
----------┴-------------------------------------------------------------------------------┘

size: 数据部分总长度，及 optional 部分

some fields: [name, mime, last_modified_date, ttl, pairs]

注意:
    如果, len(data) = 0, 则 size = 0;
    否则, size = len(data) +
                (1)(flag) +
                (4)(data_size) +
                (1+name_size)(if name exists) +
                (1+mime_size)(if mime exists) +
                (8)(if last modified date exists) + /*使用后5个字节*/
                (2)(if ttl exists) +
                (2+pairs)(if pairs exists) +
                (4)(checksum) +
                (8)(current time stamp) +
                (n)(padding, 8Bytes alignment)
```

### Filer

#### File ID

``` go
type FileId struct {
    VolumeId VolumeId       // uint32, 等同于 idx/dat 文件名
    Key      types.NeedleId // uint64
    Cookie   types.Cookie   // uint32
}

func (t *Topology) PickForWrite(count uint64, option *VolumeGrowOption) (string, uint64, *DataNode, error) {
    vid, count, datanodes, err := t.GetVolumeLayout(option.Collection, option.ReplicaPlacement, option.Ttl).PickForWrite(count, option)
    if err != nil || datanodes.Length() == 0 {
        return "", 0, nil, errors.New("No writable volumes available!")
    }
    // FIXME: fileId 实际是 needleId
    fileId, count := t.Sequence.NextFileId(count)
    return storage.NewFileId(*vid, fileId, rand.Uint32()).String(), count, datanodes.Head(), nil
}

```


