<template>
  <OuiStack gap="none">
    <!-- Loading State -->
    <OuiStack v-if="schemaLoading" align="center" gap="md" class="py-10">
      <OuiSpinner size="lg" />
      <OuiText color="secondary">Loading schema...</OuiText>
    </OuiStack>

    <!-- Error State -->
    <ErrorAlert
      v-else-if="schemaError"
      :error="schemaError"
      title="Failed to load schema"
    />

    <!-- Main browser -->
    <div v-else class="flex" style="min-height: 500px">
      <!-- Left pane: Schema tree -->
      <div
        class="border-r border-border-default shrink-0 overflow-y-auto bg-surface-base"
        :style="{ width: treePaneWidth + 'px' }"
      >
        <div class="p-3">
          <OuiFlex justify="between" align="center" class="mb-3">
            <OuiText size="xs" weight="semibold" transform="uppercase" color="secondary">
              Schema
            </OuiText>
            <OuiFlex gap="xs">
              <OuiButton variant="ghost" color="primary" size="sm" @click="openCreateTableDialog" title="Create Table">
                <PlusIcon class="h-3.5 w-3.5" />
              </OuiButton>
              <OuiButton variant="ghost" color="secondary" size="sm" @click="refreshSchema" title="Refresh Schema">
                <ArrowPathIcon class="h-3.5 w-3.5" />
              </OuiButton>
            </OuiFlex>
          </OuiFlex>

          <OuiInput
            v-model="searchQuery"
            placeholder="Filter..."
            clearable
            size="sm"
            class="mb-3"
          >
            <template #prefix>
              <MagnifyingGlassIcon class="h-3.5 w-3.5 text-secondary" />
            </template>
          </OuiInput>

          <!-- Tables section -->
          <div class="mb-3">
            <button
              class="flex items-center gap-1 w-full text-left text-xs font-semibold text-secondary hover:text-primary py-1"
              @click="showTables = !showTables"
            >
              <ChevronRightIcon
                class="h-3 w-3 transition-transform"
                :class="{ 'rotate-90': showTables }"
              />
              Tables ({{ filteredTables.length }})
            </button>
            <div v-if="showTables" class="ml-2">
              <div v-for="table in filteredTables" :key="table.name" class="mb-0.5">
                <button
                  class="flex items-center gap-1.5 w-full text-left px-2 py-1 text-xs rounded hover:bg-interactive-hover transition-colors group"
                  :class="{
                    'bg-primary/10 text-primary': selectedTableName === table.name,
                  }"
                  @click="selectTable(table)"
                  @contextmenu.prevent="showTableContextMenu($event, table)"
                >
                  <ChevronRightIcon
                    class="h-3 w-3 shrink-0 transition-transform"
                    :class="{ 'rotate-90': expandedTables.has(table.name) }"
                    @click.stop="toggleTableExpand(table.name)"
                  />
                  <TableCellsIcon class="h-3.5 w-3.5 shrink-0 text-secondary" />
                  <span class="truncate flex-1">{{ table.name }}</span>
                  <span class="text-secondary text-[10px] opacity-0 group-hover:opacity-100">
                    {{ Number(table.rowCount) }}
                  </span>
                </button>
                <!-- Expanded columns -->
                <div
                  v-if="expandedTables.has(table.name)"
                  class="ml-6 border-l border-border-default/50 pl-2"
                >
                  <div
                    v-for="col in table.columns"
                    :key="col.name"
                    class="flex items-center gap-1.5 py-0.5 text-[11px]"
                  >
                    <span
                      v-if="col.isPrimaryKey"
                      class="text-warning font-bold"
                      title="Primary Key"
                    >PK</span>
                    <span
                      v-else-if="isForeignKey(table, col.name)"
                      class="text-info font-bold"
                      title="Foreign Key"
                    >FK</span>
                    <span v-else class="w-4" />
                    <span class="truncate">{{ col.name }}</span>
                    <span class="text-secondary ml-auto text-[10px]">{{ col.dataType }}</span>
                    <span
                      v-if="col.isNullable"
                      class="text-secondary text-[9px]"
                      title="Nullable"
                    >?</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Views section -->
          <div v-if="schemaViews.length > 0" class="mb-3">
            <button
              class="flex items-center gap-1 w-full text-left text-xs font-semibold text-secondary hover:text-primary py-1"
              @click="showViews = !showViews"
            >
              <ChevronRightIcon
                class="h-3 w-3 transition-transform"
                :class="{ 'rotate-90': showViews }"
              />
              Views ({{ schemaViews.length }})
            </button>
            <div v-if="showViews" class="ml-2">
              <div
                v-for="view in schemaViews"
                :key="view.name"
                class="flex items-center gap-1.5 px-2 py-1 text-xs text-secondary"
              >
                <EyeIcon class="h-3.5 w-3.5 shrink-0" />
                <span class="truncate">{{ view.name }}</span>
              </div>
            </div>
          </div>

          <!-- Functions section -->
          <div v-if="schemaFunctions.length > 0">
            <button
              class="flex items-center gap-1 w-full text-left text-xs font-semibold text-secondary hover:text-primary py-1"
              @click="showFunctions = !showFunctions"
            >
              <ChevronRightIcon
                class="h-3 w-3 transition-transform"
                :class="{ 'rotate-90': showFunctions }"
              />
              Functions ({{ schemaFunctions.length }})
            </button>
            <div v-if="showFunctions" class="ml-2">
              <div
                v-for="fn in schemaFunctions"
                :key="fn.name"
                class="flex items-center gap-1.5 px-2 py-1 text-xs text-secondary"
              >
                <CodeBracketIcon class="h-3.5 w-3.5 shrink-0" />
                <span class="truncate">{{ fn.name }}</span>
                <span class="ml-auto text-[10px]">{{ fn.returnType }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Resize handle -->
      <div
        class="w-1 cursor-col-resize bg-transparent hover:bg-primary/20 transition-colors shrink-0"
        @mousedown="startTreeResize"
      />

      <!-- Right pane: Data / Structure -->
      <div class="flex-1 overflow-hidden flex flex-col min-w-0">
        <template v-if="selectedTableName">
          <!-- Tab bar -->
          <div class="flex items-center border-b border-border-default bg-surface-base px-3">
            <button
              v-for="tab in dataTabs"
              :key="tab.id"
              class="px-3 py-2 text-xs font-medium border-b-2 transition-colors -mb-px"
              :class="
                activeDataTab === tab.id
                  ? 'border-primary text-primary'
                  : 'border-transparent text-secondary hover:text-primary'
              "
              @click="activeDataTab = tab.id"
            >
              {{ tab.label }}
            </button>
            <div class="ml-auto flex items-center gap-2">
              <OuiText size="xs" color="secondary">
                {{ selectedTableName }}
              </OuiText>
            </div>
          </div>

          <!-- Data tab -->
          <div v-if="activeDataTab === 'data'" class="flex-1 flex flex-col overflow-hidden">
            <!-- Data toolbar -->
            <div class="flex items-center gap-2 px-3 py-2 border-b border-border-default bg-surface-base">
              <OuiButton
                variant="ghost"
                color="secondary"
                size="sm"
                @click="loadTableData"
                :loading="dataLoading"
              >
                <ArrowPathIcon class="h-3.5 w-3.5" />
                Refresh
              </OuiButton>
              <OuiButton
                v-if="pendingEdits.size > 0"
                color="primary"
                size="sm"
                @click="saveEdits"
                :loading="savingEdits"
              >
                Save {{ pendingEdits.size }} change(s)
              </OuiButton>
              <OuiButton
                v-if="pendingEdits.size > 0"
                variant="ghost"
                color="secondary"
                size="sm"
                @click="discardEdits"
              >
                Discard
              </OuiButton>
              <div class="ml-auto flex items-center gap-2">
                <OuiButton
                  variant="ghost"
                  color="success"
                  size="sm"
                  @click="startInsertRow"
                >
                  <PlusIcon class="h-3.5 w-3.5" />
                  Add Row
                </OuiButton>
                <OuiText size="xs" color="secondary">
                  {{ dataResponse?.totalRows ?? 0 }} total rows
                </OuiText>
              </div>
            </div>

            <!-- Data grid -->
            <div class="flex-1 overflow-auto">
              <table v-if="dataResponse" class="w-full text-xs">
                <thead class="sticky top-0 z-10">
                  <tr class="bg-surface-base border-b border-border-default">
                    <th class="px-2 py-1.5 text-left w-8">#</th>
                    <th
                      v-for="col in dataResponse.columns"
                      :key="col.name"
                      class="px-2 py-1.5 text-left font-medium cursor-pointer hover:bg-interactive-hover select-none whitespace-nowrap"
                      @click="toggleDataSort(col.name)"
                    >
                      <span>{{ col.name }}</span>
                      <span class="text-secondary font-normal ml-1">{{ col.dataType }}</span>
                      <span v-if="dataSortColumn === col.name" class="text-primary ml-0.5">
                        {{ dataSortDirection === 'ASC' ? '↑' : '↓' }}
                      </span>
                    </th>
                    <th class="px-2 py-1.5 w-10" />
                  </tr>
                </thead>
                <tbody>
                  <!-- Insert row -->
                  <tr v-if="insertingRow" class="bg-success/5 border-b border-border-default">
                    <td class="px-2 py-1 text-secondary">+</td>
                    <td
                      v-for="col in dataResponse.columns"
                      :key="col.name"
                      class="px-2 py-0"
                    >
                      <input
                        v-model="newRowValues[col.name]"
                        class="w-full bg-transparent border-b border-border-default text-xs py-1 px-0 focus:outline-none focus:border-primary"
                        :placeholder="col.name"
                      />
                    </td>
                    <td class="px-2 py-1">
                      <OuiFlex gap="xs">
                        <button
                          class="text-success hover:text-success/80 text-xs"
                          @click="confirmInsertRow"
                        >Save</button>
                        <button
                          class="text-secondary hover:text-danger text-xs"
                          @click="insertingRow = false"
                        >Cancel</button>
                      </OuiFlex>
                    </td>
                  </tr>

                  <!-- Data rows -->
                  <tr
                    v-for="(row, rowIdx) in dataRows"
                    :key="rowIdx"
                    class="border-b border-border-default/30 hover:bg-interactive-hover/30"
                  >
                    <td class="px-2 py-1 text-secondary font-mono">
                      {{ (dataPage - 1) * dataPerPage + rowIdx + 1 }}
                    </td>
                    <td
                      v-for="col in dataResponse.columns"
                      :key="col.name"
                      class="px-2 py-0 font-mono whitespace-nowrap max-w-xs"
                      :class="{
                        'bg-warning/10': hasEdit(rowIdx, col.name),
                      }"
                      @dblclick="startCellEdit(rowIdx, col.name, row[col.name])"
                    >
                      <!-- Editing -->
                      <input
                        v-if="editingCell?.row === rowIdx && editingCell?.col === col.name"
                        ref="editInput"
                        v-model="editingCell.value"
                        class="w-full bg-transparent border-b border-primary text-xs py-1 px-0 focus:outline-none"
                        @keydown.enter="confirmCellEdit"
                        @keydown.escape="cancelCellEdit"
                        @blur="confirmCellEdit"
                      />
                      <!-- Display -->
                      <template v-else>
                        <span
                          v-if="row[col.name] === null"
                          class="text-secondary italic"
                        >NULL</span>
                        <span v-else class="truncate block">{{ row[col.name] }}</span>
                      </template>
                    </td>
                    <td class="px-2 py-1">
                      <button
                        class="text-secondary hover:text-danger text-xs opacity-0 group-hover:opacity-100"
                        title="Delete row"
                        @click="deleteRow(rowIdx)"
                      >
                        <TrashIcon class="h-3.5 w-3.5" />
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>

              <!-- Loading -->
              <OuiStack v-if="dataLoading" align="center" gap="sm" class="py-8">
                <OuiSpinner />
                <OuiText color="secondary" size="xs">Loading data...</OuiText>
              </OuiStack>

              <!-- No data -->
              <OuiStack
                v-else-if="!dataResponse || dataRows.length === 0"
                align="center"
                gap="sm"
                class="py-8"
              >
                <OuiText color="secondary" size="sm">No data</OuiText>
              </OuiStack>
            </div>

            <!-- Pagination -->
            <div
              v-if="dataResponse && dataResponse.totalRows > dataPerPage"
              class="flex items-center justify-between px-3 py-2 border-t border-border-default bg-surface-base"
            >
              <OuiFlex gap="sm" align="center">
                <OuiButton
                  variant="ghost"
                  size="sm"
                  :disabled="dataPage <= 1"
                  @click="dataPage--; loadTableData()"
                >
                  Previous
                </OuiButton>
                <OuiText size="xs" color="secondary">
                  Page {{ dataPage }} of {{ Math.ceil(dataResponse.totalRows / dataPerPage) }}
                </OuiText>
                <OuiButton
                  variant="ghost"
                  size="sm"
                  :disabled="dataPage >= Math.ceil(dataResponse.totalRows / dataPerPage)"
                  @click="dataPage++; loadTableData()"
                >
                  Next
                </OuiButton>
              </OuiFlex>
              <OuiFlex gap="sm" align="center">
                <OuiText size="xs" color="secondary">Per page:</OuiText>
                <select
                  v-model.number="dataPerPage"
                  class="text-xs bg-transparent border border-border-default rounded px-1 py-0.5"
                  @change="dataPage = 1; loadTableData()"
                >
                  <option :value="25">25</option>
                  <option :value="50">50</option>
                  <option :value="100">100</option>
                  <option :value="200">200</option>
                </select>
              </OuiFlex>
            </div>
          </div>

          <!-- Structure tab -->
          <div v-else-if="activeDataTab === 'structure'" class="flex-1 overflow-hidden flex flex-col">
            <!-- Structure toolbar -->
            <div class="flex items-center gap-2 px-3 py-2 border-b border-border-default bg-surface-base">
              <OuiButton
                variant="ghost"
                color="primary"
                size="sm"
                @click="showAddColumn = true"
              >
                <PlusIcon class="h-3.5 w-3.5" />
                Add Column
              </OuiButton>
            </div>
            <div class="flex-1 overflow-auto p-4">
              <table v-if="selectedTable" class="w-full text-xs">
                <thead>
                  <tr class="border-b border-border-default">
                    <th class="px-3 py-2 text-left font-medium">Column</th>
                    <th class="px-3 py-2 text-left font-medium">Type</th>
                    <th class="px-3 py-2 text-left font-medium">Nullable</th>
                    <th class="px-3 py-2 text-left font-medium">Default</th>
                    <th class="px-3 py-2 text-left font-medium">PK</th>
                    <th class="px-3 py-2 text-left font-medium">Unique</th>
                    <th class="px-3 py-2 text-left font-medium w-16">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="col in selectedTable.columns"
                    :key="col.name"
                    class="border-b border-border-default/30 hover:bg-interactive-hover/30 group"
                  >
                    <td class="px-3 py-1.5 font-medium">{{ col.name }}</td>
                    <td class="px-3 py-1.5 font-mono text-secondary">{{ col.dataType }}</td>
                    <td class="px-3 py-1.5">
                      <OuiBadge :color="col.isNullable ? 'secondary' : 'warning'" size="xs">
                        {{ col.isNullable ? 'Yes' : 'No' }}
                      </OuiBadge>
                    </td>
                    <td class="px-3 py-1.5 text-secondary">{{ col.defaultValue || '—' }}</td>
                    <td class="px-3 py-1.5">
                      <OuiBadge v-if="col.isPrimaryKey" color="primary" size="xs">PK</OuiBadge>
                      <span v-else class="text-secondary">—</span>
                    </td>
                    <td class="px-3 py-1.5">
                      <OuiBadge v-if="col.isUnique" color="info" size="xs">Unique</OuiBadge>
                      <span v-else class="text-secondary">—</span>
                    </td>
                    <td class="px-3 py-1.5">
                      <button
                        v-if="!col.isPrimaryKey"
                        class="text-secondary hover:text-danger opacity-0 group-hover:opacity-100 transition-opacity"
                        title="Drop column"
                        @click="dropColumn(col.name)"
                      >
                        <TrashIcon class="h-3.5 w-3.5" />
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Indexes tab -->
          <div v-else-if="activeDataTab === 'indexes'" class="flex-1 overflow-hidden flex flex-col">
            <!-- Indexes toolbar -->
            <div class="flex items-center gap-2 px-3 py-2 border-b border-border-default bg-surface-base">
              <OuiButton
                variant="ghost"
                color="primary"
                size="sm"
                @click="showCreateIndex = true"
              >
                <PlusIcon class="h-3.5 w-3.5" />
                Create Index
              </OuiButton>
            </div>
            <div class="flex-1 overflow-auto p-4">
              <table v-if="selectedTable && selectedTable.indexes.length > 0" class="w-full text-xs">
                <thead>
                  <tr class="border-b border-border-default">
                    <th class="px-3 py-2 text-left font-medium">Name</th>
                    <th class="px-3 py-2 text-left font-medium">Columns</th>
                    <th class="px-3 py-2 text-left font-medium">Type</th>
                    <th class="px-3 py-2 text-left font-medium">Unique</th>
                    <th class="px-3 py-2 text-left font-medium">Primary</th>
                    <th class="px-3 py-2 text-left font-medium w-16">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="idx in selectedTable.indexes"
                    :key="idx.name"
                    class="border-b border-border-default/30 hover:bg-interactive-hover/30 group"
                  >
                    <td class="px-3 py-1.5 font-medium">{{ idx.name }}</td>
                    <td class="px-3 py-1.5 font-mono text-secondary">{{ idx.columnNames.join(', ') }}</td>
                    <td class="px-3 py-1.5 text-secondary">{{ idx.type || '—' }}</td>
                    <td class="px-3 py-1.5">
                      <OuiBadge v-if="idx.isUnique" color="info" size="xs">Yes</OuiBadge>
                      <span v-else class="text-secondary">No</span>
                    </td>
                    <td class="px-3 py-1.5">
                      <OuiBadge v-if="idx.isPrimary" color="primary" size="xs">Yes</OuiBadge>
                      <span v-else class="text-secondary">No</span>
                    </td>
                    <td class="px-3 py-1.5">
                      <button
                        v-if="!idx.isPrimary"
                        class="text-secondary hover:text-danger opacity-0 group-hover:opacity-100 transition-opacity"
                        title="Drop index"
                        @click="dropIndex(idx.name)"
                      >
                        <TrashIcon class="h-3.5 w-3.5" />
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
              <OuiText v-else color="secondary" size="sm" class="py-4 text-center">
                No indexes found
              </OuiText>
            </div>
          </div>

          <!-- Foreign Keys tab -->
          <div v-else-if="activeDataTab === 'foreignKeys'" class="flex-1 overflow-auto p-4">
            <table v-if="selectedTable && selectedTable.foreignKeys.length > 0" class="w-full text-xs">
              <thead>
                <tr class="border-b border-border-default">
                  <th class="px-3 py-2 text-left font-medium">Name</th>
                  <th class="px-3 py-2 text-left font-medium">From</th>
                  <th class="px-3 py-2 text-left font-medium">To</th>
                  <th class="px-3 py-2 text-left font-medium">On Delete</th>
                  <th class="px-3 py-2 text-left font-medium">On Update</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="fk in selectedTable.foreignKeys"
                  :key="fk.name"
                  class="border-b border-border-default/30 hover:bg-interactive-hover/30"
                >
                  <td class="px-3 py-1.5 font-medium">{{ fk.name }}</td>
                  <td class="px-3 py-1.5 font-mono text-secondary">{{ fk.fromColumns.join(', ') }}</td>
                  <td class="px-3 py-1.5 font-mono">
                    <span class="text-primary">{{ fk.toTable }}</span>.{{ fk.toColumns.join(', ') }}
                  </td>
                  <td class="px-3 py-1.5 text-secondary">{{ fk.onDelete || '—' }}</td>
                  <td class="px-3 py-1.5 text-secondary">{{ fk.onUpdate || '—' }}</td>
                </tr>
              </tbody>
            </table>
            <OuiText v-else color="secondary" size="sm" class="py-4 text-center">
              No foreign keys found
            </OuiText>
          </div>

          <!-- DDL tab -->
          <div v-else-if="activeDataTab === 'ddl'" class="flex-1 overflow-hidden flex flex-col">
            <!-- DDL toolbar -->
            <div class="flex items-center gap-2 px-3 py-2 border-b border-border-default bg-surface-base">
              <OuiButton
                variant="ghost"
                color="secondary"
                size="sm"
                @click="loadTableDDL"
                :loading="loadingDDL"
              >
                <ArrowPathIcon class="h-3.5 w-3.5" />
                Refresh
              </OuiButton>
              <OuiButton
                variant="ghost"
                color="secondary"
                size="sm"
                @click="copyDDL"
                :disabled="!tableDDL"
              >
                <ClipboardDocumentIcon class="h-3.5 w-3.5" />
                Copy
              </OuiButton>
            </div>
            <div class="flex-1 overflow-auto p-4">
              <OuiStack v-if="loadingDDL" align="center" gap="sm" class="py-8">
                <OuiSpinner />
                <OuiText color="secondary" size="xs">Loading DDL...</OuiText>
              </OuiStack>
              <pre
                v-else-if="tableDDL"
                class="text-xs font-mono bg-surface-base border border-border-default rounded-lg p-4 overflow-x-auto whitespace-pre-wrap"
              >{{ tableDDL }}</pre>
              <OuiText v-else color="secondary" size="sm" class="py-4 text-center">
                No DDL available
              </OuiText>
            </div>
          </div>
        </template>

        <!-- No table selected -->
        <OuiStack v-else align="center" justify="center" class="flex-1 py-16">
          <TableCellsIcon class="h-12 w-12 text-secondary/30" />
          <OuiText color="secondary" size="sm">Select a table to browse</OuiText>
        </OuiStack>
      </div>
    </div>

    <!-- Context Menu -->
    <Teleport to="body">
      <div
        v-if="contextMenu"
        class="fixed z-50 bg-surface-overlay border border-border-default rounded-lg shadow-lg py-1 min-w-40"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
        @click.stop
      >
        <button
          class="w-full text-left px-3 py-1.5 text-xs hover:bg-interactive-hover flex items-center gap-2"
          @click="contextMenuAction('viewData')"
        >
          <TableCellsIcon class="h-3.5 w-3.5 text-secondary" />
          View Data
        </button>
        <button
          class="w-full text-left px-3 py-1.5 text-xs hover:bg-interactive-hover flex items-center gap-2"
          @click="contextMenuAction('editStructure')"
        >
          <WrenchIcon class="h-3.5 w-3.5 text-secondary" />
          Edit Structure
        </button>
        <button
          class="w-full text-left px-3 py-1.5 text-xs hover:bg-interactive-hover flex items-center gap-2"
          @click="contextMenuAction('viewDDL')"
        >
          <CodeBracketIcon class="h-3.5 w-3.5 text-secondary" />
          View DDL
        </button>
        <div class="border-t border-border-default my-1" />
        <button
          class="w-full text-left px-3 py-1.5 text-xs hover:bg-interactive-hover flex items-center gap-2"
          @click="contextMenuAction('rename')"
        >
          <PencilIcon class="h-3.5 w-3.5 text-secondary" />
          Rename Table
        </button>
        <button
          class="w-full text-left px-3 py-1.5 text-xs hover:bg-interactive-hover text-warning flex items-center gap-2"
          @click="contextMenuAction('truncate')"
        >
          <ExclamationTriangleIcon class="h-3.5 w-3.5" />
          Truncate Table
        </button>
        <button
          class="w-full text-left px-3 py-1.5 text-xs hover:bg-interactive-hover text-danger flex items-center gap-2"
          @click="contextMenuAction('drop')"
        >
          <TrashIcon class="h-3.5 w-3.5" />
          Drop Table
        </button>
      </div>
    </Teleport>

    <!-- Add Column Dialog -->
    <OuiDialog v-model:open="showAddColumn" title="Add Column" size="sm">
      <OuiStack gap="md">
        <OuiFormField label="Column Name" required>
          <OuiInput v-model="newColumn.name" placeholder="column_name" />
        </OuiFormField>
        <OuiFormField label="Data Type" required>
          <select
            v-model="newColumn.dataType"
            class="w-full bg-surface-base border border-border-default rounded-md px-3 py-2 text-sm"
          >
            <optgroup label="String">
              <option value="varchar(255)">varchar(255)</option>
              <option value="text">text</option>
              <option value="char(1)">char(1)</option>
              <option value="uuid">uuid</option>
            </optgroup>
            <optgroup label="Numeric">
              <option value="integer">integer</option>
              <option value="bigint">bigint</option>
              <option value="smallint">smallint</option>
              <option value="decimal">decimal</option>
              <option value="numeric">numeric</option>
              <option value="real">real</option>
              <option value="double precision">double precision</option>
            </optgroup>
            <optgroup label="Date/Time">
              <option value="timestamp">timestamp</option>
              <option value="timestamptz">timestamptz</option>
              <option value="date">date</option>
              <option value="time">time</option>
            </optgroup>
            <optgroup label="Other">
              <option value="boolean">boolean</option>
              <option value="jsonb">jsonb</option>
              <option value="json">json</option>
              <option value="bytea">bytea</option>
            </optgroup>
          </select>
        </OuiFormField>
        <OuiFlex gap="md">
          <label class="flex items-center gap-2 text-sm cursor-pointer">
            <input type="checkbox" v-model="newColumn.isNullable" class="rounded" />
            Nullable
          </label>
          <label class="flex items-center gap-2 text-sm cursor-pointer">
            <input type="checkbox" v-model="newColumn.isUnique" class="rounded" />
            Unique
          </label>
        </OuiFlex>
        <OuiFormField label="Default Value">
          <OuiInput v-model="newColumn.defaultValue" placeholder="NULL" />
        </OuiFormField>
      </OuiStack>
      <template #footer>
        <OuiButton variant="ghost" @click="showAddColumn = false">Cancel</OuiButton>
        <OuiButton color="primary" @click="addColumn" :disabled="!newColumn.name">Add Column</OuiButton>
      </template>
    </OuiDialog>

    <!-- Create Index Dialog -->
    <OuiDialog v-model:open="showCreateIndex" title="Create Index" size="sm">
      <OuiStack gap="md">
        <OuiFormField label="Index Name" required>
          <OuiInput v-model="newIndex.name" :placeholder="`idx_${selectedTableName}_`" />
        </OuiFormField>
        <OuiFormField label="Columns" required>
          <div class="space-y-1 max-h-40 overflow-y-auto border border-border-default rounded-md p-2">
            <label
              v-for="col in selectedTable?.columns || []"
              :key="col.name"
              class="flex items-center gap-2 text-sm cursor-pointer py-0.5 hover:bg-interactive-hover px-1 rounded"
            >
              <input
                type="checkbox"
                :checked="newIndex.columnNames.includes(col.name)"
                @change="toggleIndexColumn(col.name)"
                class="rounded"
              />
              <span>{{ col.name }}</span>
              <span class="text-secondary text-xs ml-auto">{{ col.dataType }}</span>
            </label>
          </div>
        </OuiFormField>
        <label class="flex items-center gap-2 text-sm cursor-pointer">
          <input type="checkbox" v-model="newIndex.isUnique" class="rounded" />
          Unique Index
        </label>
      </OuiStack>
      <template #footer>
        <OuiButton variant="ghost" @click="showCreateIndex = false">Cancel</OuiButton>
        <OuiButton
          color="primary"
          @click="createIndex"
          :disabled="!newIndex.name || newIndex.columnNames.length === 0"
        >Create Index</OuiButton>
      </template>
    </OuiDialog>

    <!-- Rename Table Dialog -->
    <OuiDialog v-model:open="showRenameTable" title="Rename Table" size="sm">
      <OuiStack gap="md">
        <OuiFormField label="New Table Name" required>
          <OuiInput v-model="renameTableName" />
        </OuiFormField>
      </OuiStack>
      <template #footer>
        <OuiButton variant="ghost" @click="showRenameTable = false">Cancel</OuiButton>
        <OuiButton color="primary" @click="renameTable" :disabled="!renameTableName">Rename</OuiButton>
      </template>
    </OuiDialog>

    <!-- Create Table Dialog -->
    <DatabaseTableDesigner
      v-if="showCreateTable"
      v-model:open="showCreateTable"
      :database-id="databaseId"
      :database-type="databaseType"
      @created="refreshSchema"
    />
  </OuiStack>
</template>

<script setup lang="ts">
import {
  MagnifyingGlassIcon,
  ArrowPathIcon,
  ChevronRightIcon,
  TableCellsIcon,
  EyeIcon,
  CodeBracketIcon,
  PlusIcon,
  TrashIcon,
  WrenchIcon,
  PencilIcon,
  ExclamationTriangleIcon,
  ClipboardDocumentIcon,
} from "@heroicons/vue/24/outline";
import { ref, computed, onMounted, onUnmounted, nextTick, toRef, watch } from "vue";
import { DatabaseService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useToast } from "~/composables/useToast";
import { useDialog } from "~/composables/useDialog";
import { useDatabaseSchema, type SchemaTable } from "~/composables/useDatabaseSchema";
import ErrorAlert from "~/components/ErrorAlert.vue";

const props = defineProps<{
  databaseId: string;
  databaseType: string;
}>();

const organizationId = useOrganizationId();
const { toast } = useToast();
const { showConfirm } = useDialog();
const dbClient = useConnectClient(DatabaseService);

// Schema
const {
  tables: schemaTables,
  views: schemaViews,
  functions: schemaFunctions,
  loading: schemaLoading,
  error: schemaError,
  fetchSchema,
  refresh: refreshSchema,
} = useDatabaseSchema(toRef(props, "databaseId"));

// Tree state
const searchQuery = ref("");
const showTables = ref(true);
const showViews = ref(false);
const showFunctions = ref(false);
const expandedTables = ref(new Set<string>());
const selectedTableName = ref<string | null>(null);

// Tree pane resize
const treePaneWidth = ref(260);
let treeResizing = false;
let treeStartX = 0;
let treeStartWidth = 0;

function startTreeResize(e: MouseEvent) {
  treeResizing = true;
  treeStartX = e.clientX;
  treeStartWidth = treePaneWidth.value;
  document.addEventListener("mousemove", onTreeResize);
  document.addEventListener("mouseup", stopTreeResize);
  e.preventDefault();
}

function onTreeResize(e: MouseEvent) {
  if (!treeResizing) return;
  const delta = e.clientX - treeStartX;
  treePaneWidth.value = Math.max(180, Math.min(500, treeStartWidth + delta));
}

function stopTreeResize() {
  treeResizing = false;
  document.removeEventListener("mousemove", onTreeResize);
  document.removeEventListener("mouseup", stopTreeResize);
}

// Data tab state
const dataTabs = [
  { id: "data", label: "Data" },
  { id: "structure", label: "Structure" },
  { id: "indexes", label: "Indexes" },
  { id: "foreignKeys", label: "Foreign Keys" },
  { id: "ddl", label: "DDL" },
];
const activeDataTab = ref("data");

// Context menu
const contextMenu = ref<{ x: number; y: number; table: SchemaTable } | null>(null);

// DDL state
const tableDDL = ref<string>("");
const loadingDDL = ref(false);

// Create table dialog
const showCreateTable = ref(false);

// Add column dialog
const showAddColumn = ref(false);
const newColumn = ref({
  name: "",
  dataType: "varchar(255)",
  isNullable: true,
  defaultValue: "",
  isUnique: false,
});

// Create index dialog
const showCreateIndex = ref(false);
const newIndex = ref({
  name: "",
  columnNames: [] as string[],
  isUnique: false,
});

// Rename table dialog
const showRenameTable = ref(false);
const renameTableName = ref("");

// Data loading
const dataLoading = ref(false);
const dataResponse = ref<any>(null);
const dataRows = ref<Record<string, any>[]>([]);
const dataPage = ref(1);
const dataPerPage = ref(50);
const dataSortColumn = ref<string | null>(null);
const dataSortDirection = ref<"ASC" | "DESC">("ASC");

// Inline editing
const editingCell = ref<{ row: number; col: string; value: string } | null>(null);
const pendingEdits = ref(new Map<string, { rowIdx: number; col: string; oldValue: any; newValue: string }>());
const savingEdits = ref(false);

// Insert row
const insertingRow = ref(false);
const newRowValues = ref<Record<string, string>>({});

const filteredTables = computed(() => {
  if (!searchQuery.value) return schemaTables.value;
  const q = searchQuery.value.toLowerCase();
  return schemaTables.value.filter((t) => t.name.toLowerCase().includes(q));
});

const selectedTable = computed(() => {
  if (!selectedTableName.value) return null;
  return schemaTables.value.find((t) => t.name === selectedTableName.value) || null;
});

function toggleTableExpand(name: string) {
  if (expandedTables.value.has(name)) {
    expandedTables.value.delete(name);
  } else {
    expandedTables.value.add(name);
  }
}

function isForeignKey(table: SchemaTable, colName: string): boolean {
  return table.foreignKeys.some((fk) => fk.fromColumns.includes(colName));
}

function selectTable(table: SchemaTable) {
  selectedTableName.value = table.name;
  activeDataTab.value = "data";
  dataPage.value = 1;
  dataSortColumn.value = null;
  pendingEdits.value.clear();
  editingCell.value = null;
  insertingRow.value = false;
  loadTableData();
}

// Data sort
function toggleDataSort(colName: string) {
  if (dataSortColumn.value === colName) {
    dataSortDirection.value = dataSortDirection.value === "ASC" ? "DESC" : "ASC";
  } else {
    dataSortColumn.value = colName;
    dataSortDirection.value = "ASC";
  }
  dataPage.value = 1;
  loadTableData();
}

// Load table data
async function loadTableData() {
  if (!selectedTableName.value || !organizationId.value) return;

  dataLoading.value = true;
  try {
    const res = await dbClient.getTableData({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      tableName: selectedTableName.value,
      page: dataPage.value,
      perPage: dataPerPage.value,
      sortColumn: dataSortColumn.value || undefined,
      sortDirection: dataSortColumn.value ? dataSortDirection.value : undefined,
    });

    dataResponse.value = res;
    dataRows.value = (res.rows || []).map((row: any) => {
      const obj: Record<string, any> = {};
      for (const cell of row.cells || []) {
        obj[cell.columnName] = cell.isNull ? null : cell.value;
      }
      return obj;
    });
  } catch (err: any) {
    toast.error("Failed to load table data", err.message);
  } finally {
    dataLoading.value = false;
  }
}

// Cell editing
function startCellEdit(rowIdx: number, colName: string, currentValue: any) {
  editingCell.value = {
    row: rowIdx,
    col: colName,
    value: currentValue === null ? "" : String(currentValue),
  };
  nextTick(() => {
    const inputs = document.querySelectorAll<HTMLInputElement>('[ref="editInput"]');
    inputs.forEach((el) => el.focus());
  });
}

function confirmCellEdit() {
  if (!editingCell.value) return;

  const { row, col, value } = editingCell.value;
  const rowData = dataRows.value[row];
  if (!rowData) return;
  const oldValue = rowData[col];
  const newValue = value;

  // Only record if changed
  if (String(oldValue ?? "") !== newValue) {
    const key = `${row}:${col}`;
    pendingEdits.value.set(key, { rowIdx: row, col, oldValue, newValue });
    // Update display
    rowData[col] = newValue === "" ? null : newValue;
  }

  editingCell.value = null;
}

function cancelCellEdit() {
  editingCell.value = null;
}

function hasEdit(rowIdx: number, colName: string): boolean {
  return pendingEdits.value.has(`${rowIdx}:${colName}`);
}

function discardEdits() {
  // Reload data to reset
  pendingEdits.value.clear();
  editingCell.value = null;
  loadTableData();
}

// Save edits
async function saveEdits() {
  if (!selectedTableName.value || !organizationId.value || !selectedTable.value) return;

  savingEdits.value = true;
  const pkColumns = selectedTable.value.columns.filter((c) => c.isPrimaryKey);

  try {
    for (const edit of pendingEdits.value.values()) {
      const row = dataRows.value[edit.rowIdx];
      if (!row) continue;

      // Build where cells from PK
      const whereCells = pkColumns.map((pk) => ({
        columnName: pk.name,
        value: row[pk.name] !== null ? String(row[pk.name]) : undefined,
        isNull: row[pk.name] === null,
      }));

      // If no PK, use all original column values (less safe but works)
      if (whereCells.length === 0) {
        toast.error("Cannot save edits: table has no primary key");
        return;
      }

      const setCells = [{
        columnName: edit.col,
        value: edit.newValue === "" ? undefined : edit.newValue,
        isNull: edit.newValue === "",
      }];

      await dbClient.updateTableRow({
        organizationId: organizationId.value,
        databaseId: props.databaseId,
        tableName: selectedTableName.value,
        whereCells,
        setCells,
      });
    }

    pendingEdits.value.clear();
    toast.success("Changes saved");
    loadTableData();
  } catch (err: any) {
    toast.error("Failed to save changes", err.message);
  } finally {
    savingEdits.value = false;
  }
}

// Insert row
function startInsertRow() {
  insertingRow.value = true;
  newRowValues.value = {};
}

async function confirmInsertRow() {
  if (!selectedTableName.value || !organizationId.value) return;

  const cells = Object.entries(newRowValues.value)
    .filter(([_, v]) => v !== "")
    .map(([col, val]) => ({
      columnName: col,
      value: val,
      isNull: false,
    }));

  if (cells.length === 0) {
    toast.error("At least one value is required");
    return;
  }

  try {
    await dbClient.insertTableRow({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      tableName: selectedTableName.value,
      cells,
    });
    insertingRow.value = false;
    newRowValues.value = {};
    toast.success("Row inserted");
    loadTableData();
  } catch (err: any) {
    toast.error("Failed to insert row", err.message);
  }
}

// Delete row
async function deleteRow(rowIdx: number) {
  if (!selectedTableName.value || !organizationId.value || !selectedTable.value) return;

  const confirmed = await showConfirm({
    title: "Delete Row",
    message: "Are you sure you want to delete this row? This cannot be undone.",
    confirmLabel: "Delete",
    cancelLabel: "Cancel",
    variant: "danger",
  });
  if (!confirmed) return;

  const pkColumns = selectedTable.value.columns.filter((c) => c.isPrimaryKey);
  if (pkColumns.length === 0) {
    toast.error("Cannot delete: table has no primary key");
    return;
  }

  const row = dataRows.value[rowIdx];
  if (!row) return;
  const whereCells = pkColumns.map((pk) => ({
    columnName: pk.name,
    value: row[pk.name] !== null ? String(row[pk.name]) : undefined,
    isNull: row[pk.name] === null,
  }));

  try {
    await dbClient.deleteTableRows({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      tableName: selectedTableName.value,
      whereCells,
    });
    toast.success("Row deleted");
    loadTableData();
  } catch (err: any) {
    toast.error("Failed to delete row", err.message);
  }
}

// Context menu
function showTableContextMenu(e: MouseEvent, table: SchemaTable) {
  contextMenu.value = { x: e.clientX, y: e.clientY, table };
}

function closeContextMenu() {
  contextMenu.value = null;
}

function contextMenuAction(action: string) {
  const table = contextMenu.value?.table;
  if (!table) return;
  closeContextMenu();

  switch (action) {
    case "viewData":
      selectTable(table);
      activeDataTab.value = "data";
      break;
    case "editStructure":
      selectTable(table);
      activeDataTab.value = "structure";
      break;
    case "viewDDL":
      selectTable(table);
      activeDataTab.value = "ddl";
      break;
    case "rename":
      selectedTableName.value = table.name;
      renameTableName.value = table.name;
      showRenameTable.value = true;
      break;
    case "truncate":
      truncateSelectedTable(table);
      break;
    case "drop":
      dropSelectedTable(table);
      break;
  }
}

// DDL operations
async function loadTableDDL() {
  if (!selectedTableName.value || !organizationId.value) return;

  loadingDDL.value = true;
  try {
    const res = await dbClient.getTableDDL({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      tableName: selectedTableName.value,
    });
    tableDDL.value = res.ddl;
  } catch (err: any) {
    toast.error("Failed to load DDL", err.message);
    tableDDL.value = "";
  } finally {
    loadingDDL.value = false;
  }
}

// Load DDL when switching to DDL tab
watch(activeDataTab, (tab) => {
  if (tab === "ddl" && selectedTableName.value) {
    loadTableDDL();
  }
});

// Copy DDL to clipboard
async function copyDDL() {
  if (!tableDDL.value) return;
  try {
    await navigator.clipboard.writeText(tableDDL.value);
    toast.success("DDL copied to clipboard");
  } catch {
    toast.error("Failed to copy to clipboard");
  }
}

// Create table dialog
function openCreateTableDialog() {
  showCreateTable.value = true;
}

// Add column
async function addColumn() {
  if (!selectedTableName.value || !organizationId.value || !newColumn.value.name) return;

  try {
    await dbClient.alterTable({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      tableName: selectedTableName.value,
      operations: [{
        operation: {
          case: "addColumn",
          value: {
            column: {
              name: newColumn.value.name,
              dataType: newColumn.value.dataType,
              isNullable: newColumn.value.isNullable,
              defaultValue: newColumn.value.defaultValue || undefined,
              isUnique: newColumn.value.isUnique,
            },
          },
        },
      }],
    });
    toast.success(`Column "${newColumn.value.name}" added`);
    showAddColumn.value = false;
    newColumn.value = { name: "", dataType: "varchar(255)", isNullable: true, defaultValue: "", isUnique: false };
    refreshSchema();
  } catch (err: any) {
    toast.error("Failed to add column", err.message);
  }
}

// Drop column
async function dropColumn(colName: string) {
  if (!selectedTableName.value || !organizationId.value) return;

  const confirmed = await showConfirm({
    title: "Drop Column",
    message: `Are you sure you want to drop column "${colName}"? This cannot be undone.`,
    confirmLabel: "Drop Column",
    cancelLabel: "Cancel",
    variant: "danger",
  });
  if (!confirmed) return;

  try {
    await dbClient.alterTable({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      tableName: selectedTableName.value,
      operations: [{
        operation: {
          case: "dropColumn",
          value: { columnName: colName, cascade: false },
        },
      }],
    });
    toast.success(`Column "${colName}" dropped`);
    refreshSchema();
  } catch (err: any) {
    toast.error("Failed to drop column", err.message);
  }
}

// Create index
async function createIndex() {
  if (!selectedTableName.value || !organizationId.value || !newIndex.value.name || newIndex.value.columnNames.length === 0) return;

  try {
    await dbClient.createIndex({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      tableName: selectedTableName.value,
      index: {
        name: newIndex.value.name,
        columnNames: newIndex.value.columnNames,
        isUnique: newIndex.value.isUnique,
      },
      ifNotExists: true,
    });
    toast.success(`Index "${newIndex.value.name}" created`);
    showCreateIndex.value = false;
    newIndex.value = { name: "", columnNames: [], isUnique: false };
    refreshSchema();
  } catch (err: any) {
    toast.error("Failed to create index", err.message);
  }
}

// Drop index
async function dropIndex(indexName: string) {
  if (!organizationId.value) return;

  const confirmed = await showConfirm({
    title: "Drop Index",
    message: `Are you sure you want to drop index "${indexName}"?`,
    confirmLabel: "Drop Index",
    cancelLabel: "Cancel",
    variant: "danger",
  });
  if (!confirmed) return;

  try {
    await dbClient.dropIndex({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      indexName,
      ifExists: true,
    });
    toast.success(`Index "${indexName}" dropped`);
    refreshSchema();
  } catch (err: any) {
    toast.error("Failed to drop index", err.message);
  }
}

// Rename table
async function renameTable() {
  if (!selectedTableName.value || !organizationId.value || !renameTableName.value) return;

  try {
    await dbClient.renameTable({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      oldName: selectedTableName.value,
      newName: renameTableName.value,
    });
    toast.success(`Table renamed to "${renameTableName.value}"`);
    showRenameTable.value = false;
    selectedTableName.value = renameTableName.value;
    refreshSchema();
  } catch (err: any) {
    toast.error("Failed to rename table", err.message);
  }
}

// Truncate table
async function truncateSelectedTable(table: SchemaTable) {
  if (!organizationId.value) return;

  const confirmed = await showConfirm({
    title: "Truncate Table",
    message: `Are you sure you want to truncate table "${table.name}"? All data will be permanently deleted.`,
    confirmLabel: "Truncate",
    cancelLabel: "Cancel",
    variant: "danger",
  });
  if (!confirmed) return;

  try {
    const res = await dbClient.truncateTable({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      tableName: table.name,
      cascade: false,
    });
    toast.success(`Table truncated (${res.rowsDeleted} rows deleted)`);
    if (selectedTableName.value === table.name) {
      loadTableData();
    }
    refreshSchema();
  } catch (err: any) {
    toast.error("Failed to truncate table", err.message);
  }
}

// Drop table
async function dropSelectedTable(table: SchemaTable) {
  if (!organizationId.value) return;

  const confirmed = await showConfirm({
    title: "Drop Table",
    message: `Are you sure you want to drop table "${table.name}"? This cannot be undone.`,
    confirmLabel: "Drop Table",
    cancelLabel: "Cancel",
    variant: "danger",
  });
  if (!confirmed) return;

  try {
    await dbClient.dropTable({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      tableName: table.name,
      cascade: false,
      ifExists: true,
    });
    toast.success(`Table "${table.name}" dropped`);
    if (selectedTableName.value === table.name) {
      selectedTableName.value = null;
      dataResponse.value = null;
      dataRows.value = [];
    }
    refreshSchema();
  } catch (err: any) {
    toast.error("Failed to drop table", err.message);
  }
}

// Toggle column in index selection
function toggleIndexColumn(colName: string) {
  const idx = newIndex.value.columnNames.indexOf(colName);
  if (idx > -1) {
    newIndex.value.columnNames.splice(idx, 1);
  } else {
    newIndex.value.columnNames.push(colName);
  }
}

onMounted(() => {
  fetchSchema();
  document.addEventListener("click", closeContextMenu);
});

// Cleanup resize listeners on unmount
onUnmounted(() => {
  document.removeEventListener("mousemove", onTreeResize);
  document.removeEventListener("mouseup", stopTreeResize);
  document.removeEventListener("click", closeContextMenu);
});
</script>
