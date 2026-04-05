package databases

import databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

func databaseSizeCatalog() []*databasesv1.DatabaseSize {
	return []*databasesv1.DatabaseSize{
		{
			Id:                 "small",
			Name:               "Small",
			Type:               databasesv1.DatabaseType_POSTGRESQL,
			CpuCores:           1,
			MemoryBytes:        2 * 1024 * 1024 * 1024,
			DiskBytes:          10 * 1024 * 1024 * 1024,
			MaxConnections:     100,
			PriceCentsPerMonth: 1000,
		},
		{
			Id:                 "medium",
			Name:               "Medium",
			Type:               databasesv1.DatabaseType_POSTGRESQL,
			CpuCores:           2,
			MemoryBytes:        4 * 1024 * 1024 * 1024,
			DiskBytes:          50 * 1024 * 1024 * 1024,
			MaxConnections:     200,
			PriceCentsPerMonth: 2000,
		},
		{
			Id:                 "large",
			Name:               "Large",
			Type:               databasesv1.DatabaseType_POSTGRESQL,
			CpuCores:           4,
			MemoryBytes:        8 * 1024 * 1024 * 1024,
			DiskBytes:          100 * 1024 * 1024 * 1024,
			MaxConnections:     500,
			PriceCentsPerMonth: 4000,
		},
		{
			Id:                 "small",
			Name:               "Small",
			Type:               databasesv1.DatabaseType_MYSQL,
			CpuCores:           1,
			MemoryBytes:        2 * 1024 * 1024 * 1024,
			DiskBytes:          10 * 1024 * 1024 * 1024,
			MaxConnections:     100,
			PriceCentsPerMonth: 1000,
		},
		{
			Id:                 "medium",
			Name:               "Medium",
			Type:               databasesv1.DatabaseType_MYSQL,
			CpuCores:           2,
			MemoryBytes:        4 * 1024 * 1024 * 1024,
			DiskBytes:          50 * 1024 * 1024 * 1024,
			MaxConnections:     200,
			PriceCentsPerMonth: 2000,
		},
		{
			Id:                 "large",
			Name:               "Large",
			Type:               databasesv1.DatabaseType_MYSQL,
			CpuCores:           4,
			MemoryBytes:        8 * 1024 * 1024 * 1024,
			DiskBytes:          100 * 1024 * 1024 * 1024,
			MaxConnections:     500,
			PriceCentsPerMonth: 4000,
		},
		{
			Id:                 "small",
			Name:               "Small",
			Type:               databasesv1.DatabaseType_MARIADB,
			CpuCores:           1,
			MemoryBytes:        2 * 1024 * 1024 * 1024,
			DiskBytes:          10 * 1024 * 1024 * 1024,
			MaxConnections:     100,
			PriceCentsPerMonth: 1000,
		},
		{
			Id:                 "medium",
			Name:               "Medium",
			Type:               databasesv1.DatabaseType_MARIADB,
			CpuCores:           2,
			MemoryBytes:        4 * 1024 * 1024 * 1024,
			DiskBytes:          50 * 1024 * 1024 * 1024,
			MaxConnections:     200,
			PriceCentsPerMonth: 2000,
		},
		{
			Id:                 "large",
			Name:               "Large",
			Type:               databasesv1.DatabaseType_MARIADB,
			CpuCores:           4,
			MemoryBytes:        8 * 1024 * 1024 * 1024,
			DiskBytes:          100 * 1024 * 1024 * 1024,
			MaxConnections:     500,
			PriceCentsPerMonth: 4000,
		},
		{
			Id:                 "small",
			Name:               "Small",
			Type:               databasesv1.DatabaseType_MONGODB,
			CpuCores:           1,
			MemoryBytes:        3 * 1024 * 1024 * 1024,
			DiskBytes:          15 * 1024 * 1024 * 1024,
			MaxConnections:     200,
			PriceCentsPerMonth: 1200,
		},
		{
			Id:                 "medium",
			Name:               "Medium",
			Type:               databasesv1.DatabaseType_MONGODB,
			CpuCores:           2,
			MemoryBytes:        6 * 1024 * 1024 * 1024,
			DiskBytes:          50 * 1024 * 1024 * 1024,
			MaxConnections:     400,
			PriceCentsPerMonth: 2400,
		},
		{
			Id:                 "large",
			Name:               "Large",
			Type:               databasesv1.DatabaseType_MONGODB,
			CpuCores:           4,
			MemoryBytes:        10 * 1024 * 1024 * 1024,
			DiskBytes:          100 * 1024 * 1024 * 1024,
			MaxConnections:     800,
			PriceCentsPerMonth: 4800,
		},
		{
			Id:                 "small",
			Name:               "Small",
			Type:               databasesv1.DatabaseType_REDIS,
			CpuCores:           1,
			MemoryBytes:        1 * 1024 * 1024 * 1024,
			DiskBytes:          4 * 1024 * 1024 * 1024,
			MaxConnections:     10000,
			PriceCentsPerMonth: 800,
		},
		{
			Id:                 "medium",
			Name:               "Medium",
			Type:               databasesv1.DatabaseType_REDIS,
			CpuCores:           2,
			MemoryBytes:        2 * 1024 * 1024 * 1024,
			DiskBytes:          8 * 1024 * 1024 * 1024,
			MaxConnections:     20000,
			PriceCentsPerMonth: 1600,
		},
		{
			Id:                 "large",
			Name:               "Large",
			Type:               databasesv1.DatabaseType_REDIS,
			CpuCores:           4,
			MemoryBytes:        4 * 1024 * 1024 * 1024,
			DiskBytes:          16 * 1024 * 1024 * 1024,
			MaxConnections:     50000,
			PriceCentsPerMonth: 3200,
		},
	}
}

func listDatabaseSizes(filterType *databasesv1.DatabaseType) []*databasesv1.DatabaseSize {
	all := databaseSizeCatalog()
	if filterType == nil || *filterType == databasesv1.DatabaseType_DATABASE_TYPE_UNSPECIFIED {
		return all
	}

	sizes := make([]*databasesv1.DatabaseSize, 0, len(all))
	for _, size := range all {
		if size.Type == *filterType {
			sizes = append(sizes, size)
		}
	}
	return sizes
}

func lookupDatabaseSize(dbType databasesv1.DatabaseType, sizeID string) (*databasesv1.DatabaseSize, bool) {
	for _, size := range databaseSizeCatalog() {
		if size.Type == dbType && size.Id == sizeID {
			return size, true
		}
	}
	return nil, false
}
