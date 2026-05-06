<template>
  <OuiStack gap="none">
    <!-- Loading State -->
    <OuiStack v-if="schemaLoading" align="center" gap="md" style="padding: 2.5rem 0">
      <OuiSpinner size="lg" />
      <OuiText color="tertiary">Loading schema...</OuiText>
    </OuiStack>

    <!-- Error State -->
    <ErrorAlert
      v-else-if="schemaError"
      :error="schemaError"
      title="Failed to load schema"
    />

    <!-- Main browser -->
    <div v-else class="db-browser-shell">
      <!-- Left pane: Schema tree -->
      <div
        class="db-schema-pane"
        :style="{ width: treePaneWidth + 'px' }"
      >
        <div class="db-schema-content">
          <OuiFlex justify="between" align="center" style="margin-bottom: 0.75rem">
            <OuiText size="xs" weight="semibold" transform="uppercase" color="tertiary">
              Schema
            </OuiText>
            <OuiFlex gap="xs">
              <OuiButton variant="ghost" color="primary" size="sm" @click="openCreateTableDialog" title="Create Table">
                <PlusIcon style="width: 0.875rem; height: 0.875rem" />
              </OuiButton>
              <OuiButton variant="ghost" color="secondary" size="sm" @click="refreshSchema" title="Refresh Schema">
                <ArrowPathIcon style="width: 0.875rem; height: 0.875rem" />
              </OuiButton>
            </OuiFlex>
          </OuiFlex>

          <OuiInput
            v-model="searchQuery"
            placeholder="Filter..."
            clearable
            size="sm"
            style="margin-bottom: 0.75rem"
          >
            <template #prefix>
              <MagnifyingGlassIcon style="width: 0.875rem; height: 0.875rem; color: var(--oui-text-secondary)" />
            </template>
          </OuiInput>

          <!-- Tables section -->
          <OuiCollapsible v-model:open="showTables" class="db-schema-collapsible">
            <template #trigger>
              <OuiFlex align="center" gap="xs" style="width: 100%; padding: 0.25rem 0">
                <OuiText size="xs" weight="semibold" color="tertiary">
                  Tables ({{ filteredTables.length }})
                </OuiText>
              </OuiFlex>
            </template>
            <div class="db-schema-list">
              <div v-for="table in filteredTables" :key="table.name" class="db-schema-table">
                <button
                  class="db-schema-table-row"
                  :class="selectedTableName === table.name ? 'bg-primary/10 text-primary' : 'bg-transparent hover:bg-surface-hover'"
                  @click="selectTable(table)"
                  @contextmenu.prevent="openContextMenu($event, table)"
                >
                  <ChevronRightIcon
                    class="w-3 h-3 shrink-0 transition-transform"
                    :class="{ 'rotate-90': expandedTables.has(table.name) }"
                    @click.stop="toggleTableExpand(table.name)"
                  />
                  <TableCellsIcon class="w-3.5 h-3.5 shrink-0 text-secondary" />
                  <span class="flex-1 truncate">{{ table.name }}</span>
                  <OuiText size="xs" color="tertiary">{{ Number(table.rowCount) }}</OuiText>
                </button>

                <!-- Expanded columns -->
                <div v-if="expandedTables.has(table.name)" class="ml-6 border-l border-border-default pl-2">
                  <OuiFlex v-for="col in table.columns" :key="col.name" align="center" gap="xs" class="py-0.5 text-[11px]">
                    <span v-if="col.isPrimaryKey" class="text-warning font-bold" title="Primary Key">PK</span>
                    <span v-else-if="isForeignKey(table, col.name)" class="text-info font-bold" title="Foreign Key">FK</span>
                    <span v-else class="w-4" />
                    <span class="truncate">{{ col.name }}</span>
                    <OuiText size="xs" color="tertiary" class="ml-auto">{{ col.dataType }}</OuiText>
                    <span v-if="col.isNullable" class="text-[9px] text-secondary" title="Nullable">?</span>
                  </OuiFlex>
                </div>
              </div>
            </div>

          </OuiCollapsible>

          <!-- Single context menu for all tables (outside collapsible) -->
          <OuiMenu v-model:open="contextMenuOpen">
            <template #trigger>
              <button
                ref="contextMenuTriggerRef"
                type="button"
                style="position: fixed; opacity: 0; pointer-events: none; width: 1px; height: 1px; z-index: -1;"
                @click.stop
              />
            </template>
            <div>
              <OuiMenuItem value="data" @select="handleTableAction('data')">
                <TableCellsIcon class="w-3.5 h-3.5 text-secondary" />
                View Data
              </OuiMenuItem>
              <OuiMenuItem value="structure" @select="handleTableAction('structure')">
                <WrenchIcon class="w-3.5 h-3.5 text-secondary" />
                Edit Structure
              </OuiMenuItem>
              <OuiMenuItem value="ddl" @select="handleTableAction('ddl')">
                <CodeBracketIcon class="w-3.5 h-3.5 text-secondary" />
                View DDL
              </OuiMenuItem>
              <OuiMenuSeparator />
              <OuiMenuItem value="rename" @select="handleTableAction('rename')">
                <PencilIcon class="w-3.5 h-3.5 text-secondary" />
                Rename Table
              </OuiMenuItem>
              <OuiMenuItem value="truncate" @select="handleTableAction('truncate')">
                <ExclamationTriangleIcon class="w-3.5 h-3.5 text-warning" />
                <span class="text-warning">Truncate Table</span>
              </OuiMenuItem>
              <OuiMenuItem value="drop" @select="handleTableAction('drop')">
                <TrashIcon class="w-3.5 h-3.5 text-danger" />
                <span class="text-danger">Drop Table</span>
              </OuiMenuItem>
            </div>
          </OuiMenu>

          <!-- Views section -->
          <OuiCollapsible v-if="schemaViews.length > 0" v-model:open="showViews" style="margin-bottom: 0.75rem">
            <template #trigger>
              <OuiFlex align="center" gap="xs" style="width: 100%; padding: 0.25rem 0">
                <OuiText size="xs" weight="semibold" color="tertiary">
                  Views ({{ schemaViews.length }})
                </OuiText>
              </OuiFlex>
            </template>
            <div style="margin-left: 0.5rem">
              <div
                v-for="view in schemaViews"
                :key="view.name"
                style="display: flex; align-items: center; gap: 0.375rem; padding: 0.25rem 0.5rem; font-size: 0.75rem; color: var(--oui-text-secondary)"
              >
                <EyeIcon style="width: 0.875rem; height: 0.875rem; flex-shrink: 0" />
                <span style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ view.name }}</span>
              </div>
            </div>
          </OuiCollapsible>

          <!-- Functions section -->
          <OuiCollapsible v-if="schemaFunctions.length > 0" v-model:open="showFunctions">
            <template #trigger>
              <OuiFlex align="center" gap="xs" style="width: 100%; padding: 0.25rem 0">
                <OuiText size="xs" weight="semibold" color="tertiary">
                  Functions ({{ schemaFunctions.length }})
                </OuiText>
              </OuiFlex>
            </template>
            <div style="margin-left: 0.5rem">
              <div
                v-for="fn in schemaFunctions"
                :key="fn.name"
                style="display: flex; align-items: center; gap: 0.375rem; padding: 0.25rem 0.5rem; font-size: 0.75rem; color: var(--oui-text-secondary)"
              >
                <CodeBracketIcon style="width: 0.875rem; height: 0.875rem; flex-shrink: 0" />
                <span style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ fn.name }}</span>
                <span style="margin-left: auto; font-size: 0.625rem">{{ fn.returnType }}</span>
              </div>
            </div>
          </OuiCollapsible>
        </div>
      </div>

      <!-- Resize handle -->
      <div
        style="width: 4px; cursor: col-resize; background: transparent; flex-shrink: 0; transition: background 0.15s"
        @mousedown="startTreeResize"
        @mouseenter="($event.target as HTMLElement).style.background = 'var(--oui-primary-alpha-20)'"
        @mouseleave="($event.target as HTMLElement).style.background = 'transparent'"
      />

      <!-- Right pane: Data / Structure -->
      <div style="flex: 1; overflow: hidden; display: flex; flex-direction: column; min-width: 0">
        <template v-if="selectedTableName">
          <div class="db-workbench-header">
            <OuiFlex justify="between" align="start" gap="md" wrap="wrap">
              <OuiStack gap="xs" class="min-w-0">
                <OuiFlex align="center" gap="sm" wrap="wrap">
                  <TableCellsIcon class="h-5 w-5 text-primary" />
                  <OuiText as="h3" size="lg" weight="semibold" class="font-mono truncate">
                    {{ selectedTableName }}
                  </OuiText>
                  <OuiBadge v-if="primaryKeyColumns.length" color="primary" size="xs">
                    PK {{ primaryKeyColumns.join(', ') }}
                  </OuiBadge>
                </OuiFlex>
                <OuiFlex gap="md" align="center" wrap="wrap">
                  <OuiText size="xs" color="tertiary">
                    {{ selectedTable?.columns.length ?? 0 }} columns
                  </OuiText>
                  <OuiText size="xs" color="tertiary">
                    {{ selectedTable?.indexes.length ?? 0 }} indexes
                  </OuiText>
                  <OuiText size="xs" color="tertiary">
                    {{ selectedTable?.foreignKeys.length ?? 0 }} foreign keys
                  </OuiText>
                  <OuiText size="xs" color="tertiary">
                    {{ dataResponse?.totalRows ?? selectedTable?.rowCount ?? 0 }} rows
                  </OuiText>
                </OuiFlex>
              </OuiStack>
              <OuiFlex gap="xs" align="center" wrap="wrap">
                <OuiButton variant="ghost" color="secondary" size="sm" @click="copyTableName">
                  <ClipboardDocumentIcon class="h-3.5 w-3.5" />
                  Copy name
                </OuiButton>
                <OuiButton variant="ghost" color="secondary" size="sm" @click="loadTableData" :loading="dataLoading">
                  <ArrowPathIcon class="h-3.5 w-3.5" />
                  Refresh
                </OuiButton>
                <OuiButton color="success" size="sm" @click="startInsertRow">
                  <PlusIcon class="h-3.5 w-3.5" />
                  Add row
                </OuiButton>
              </OuiFlex>
            </OuiFlex>
          </div>

          <OuiTabs v-model="activeDataTab" :tabs="dataTabs" content-class="p-0" style="flex: 1; display: flex; flex-direction: column; min-height: 0">
            <!-- Data tab -->
            <template #data>
              <div style="flex: 1; display: flex; flex-direction: column; overflow: hidden">
                <!-- Data toolbar -->
                <OuiFlex align="center" gap="sm" wrap="wrap" class="db-grid-toolbar">
                  <OuiInput
                    v-model="dataFilter"
                    placeholder="Search visible rows..."
                    clearable
                    size="sm"
                    class="db-grid-search"
                  >
                    <template #prefix>
                      <MagnifyingGlassIcon class="h-3.5 w-3.5 text-secondary" />
                    </template>
                  </OuiInput>
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
                  <div style="margin-left: auto; display: flex; align-items: center; gap: 0.5rem">
                    <OuiText size="xs" color="tertiary">
                      Showing {{ filteredDataRows.length }} of {{ dataRows.length }} loaded rows
                    </OuiText>
                  </div>
                </OuiFlex>

                <!-- Data grid -->
                <div class="db-grid-shell">
                  <OuiTable
                    v-if="dataResponse"
                    :columns="tableColumns"
                    :rows="filteredDataRows"
                    :sortable="true"
                    :resizable="true"
                    row-key="__rowIdx"
                    empty-text="No data"
                    wrapper-class="db-table-wrapper"
                    table-class="db-data-table"
                    header-class="db-table-header"
                    cell-class="db-table-cell"
                    @sort="handleTableSort"
                  >
                    <template #cell-__rowNum="{ row }">
                      {{ (dataPage - 1) * dataPerPage + Number(row.__rowIdx) + 1 }}
                    </template>
                    <template v-for="col in dataResponse.columns" :key="col.name" #[`header-${col.name}`]>
                      <div class="db-column-heading">
                        <span class="truncate">{{ col.name }}</span>
                        <span class="db-column-type">{{ col.dataType }}</span>
                      </div>
                    </template>
                    <template v-for="col in dataResponse.columns" :key="col.name" #[`cell-${col.name}`]="{ row }">
                      <!-- Editing -->
                      <input
                        v-if="editingCell && editingCell.row === Number(row.__rowIdx) && editingCell.col === col.name"
                        v-model="editingCell.value"
                        class="db-cell-input"
                        @keydown.enter="confirmCellEdit"
                        @keydown.escape="cancelCellEdit"
                        @blur="confirmCellEdit"
                        autofocus
                      />
                      <!-- Display -->
                      <template v-else>
                        <button
                          type="button"
                          class="db-cell-value"
                          :class="{ 'db-cell-edited': hasEdit(Number(row.__rowIdx), col.name) }"
                          :title="formatCellValue(row[col.name])"
                          @click="openCellDetail(row, col)"
                          @dblclick.stop="startCellEdit(Number(row.__rowIdx), col.name, row[col.name])"
                        >
                          <span v-if="row[col.name] === null" class="db-null">NULL</span>
                          <span v-else-if="isBooleanValue(row[col.name])" class="db-boolean">
                            <span class="db-boolean-dot" :class="isTruthyBoolean(row[col.name]) ? 'is-true' : 'is-false'" />
                            {{ String(row[col.name]) }}
                          </span>
                          <span v-else-if="isJsonColumn(col.dataType)" class="db-json">
                            {{ formatJsonPreview(row[col.name]) }}
                          </span>
                          <span v-else class="truncate">{{ row[col.name] }}</span>
                        </button>
                      </template>
                    </template>
                    <template #cell-__actions="{ row }">
                      <OuiButton variant="ghost" size="sm" color="danger" @click="deleteRow(Number(row.__rowIdx))" title="Delete row">
                        <TrashIcon style="width: 0.875rem; height: 0.875rem" />
                      </OuiButton>
                    </template>
                  </OuiTable>

                  <!-- Insert row form -->
                  <div v-if="insertingRow && dataResponse" class="db-insert-row">
                    <OuiFlex align="center" gap="sm" wrap="wrap">
                      <div v-for="col in dataResponse.columns" :key="col.name" style="min-width: 120px">
                        <OuiInput
                          v-model="newRowValues[col.name]"
                          :placeholder="col.name"
                          size="sm"
                        />
                      </div>
                      <OuiButton color="success" size="sm" @click="confirmInsertRow">Save</OuiButton>
                      <OuiButton variant="ghost" size="sm" @click="insertingRow = false">Cancel</OuiButton>
                    </OuiFlex>
                  </div>

                  <!-- Loading -->
                  <OuiStack v-if="dataLoading" align="center" gap="sm" style="padding: 2rem">
                    <OuiSpinner />
                    <OuiText color="tertiary" size="xs">Loading data...</OuiText>
                  </OuiStack>
                </div>

                <!-- Pagination -->
                <OuiFlex
                  v-if="dataResponse && dataResponse.totalRows > dataPerPage"
                  justify="between"
                  align="center"
                  style="padding: 0.5rem 0.75rem; border-top: 1px solid var(--oui-border-default); background: var(--oui-surface-base)"
                >
                  <OuiFlex gap="sm" align="center">
                    <OuiButton variant="ghost" size="sm" :disabled="dataPage <= 1" @click="dataPage--; loadTableData()">
                      Previous
                    </OuiButton>
                    <OuiText size="xs" color="tertiary">
                      Page {{ dataPage }} of {{ Math.ceil(dataResponse.totalRows / dataPerPage) }}
                    </OuiText>
                    <OuiButton variant="ghost" size="sm" :disabled="dataPage >= Math.ceil(dataResponse.totalRows / dataPerPage)" @click="dataPage++; loadTableData()">
                      Next
                    </OuiButton>
                  </OuiFlex>
                  <OuiFlex gap="sm" align="center">
                    <OuiText size="xs" color="tertiary">Per page:</OuiText>
                    <OuiSelect
                      v-model="dataPerPage"
                      :items="perPageOptions"
                      size="sm"
                      style="width: 80px"
                      @update:model-value="dataPage = 1; loadTableData()"
                    />
                  </OuiFlex>
                </OuiFlex>
              </div>
            </template>

            <!-- Structure tab -->
            <template #structure>
              <div style="flex: 1; overflow: hidden; display: flex; flex-direction: column">
                <OuiFlex align="center" gap="sm" style="padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--oui-border-default); background: var(--oui-surface-base)">
                  <OuiButton variant="ghost" color="primary" size="sm" @click="showAddColumn = true">
                    <PlusIcon style="width: 0.875rem; height: 0.875rem" />
                    Add Column
                  </OuiButton>
                </OuiFlex>
                <div style="flex: 1; overflow: auto; padding: 1rem">
                  <OuiTable
                    v-if="selectedTable"
                    :columns="structureColumns"
                    :rows="selectedTable.columns"
                    row-key="name"
                    empty-text="No columns"
                  >
                    <template #cell-name="{ value }">
                      <OuiText weight="medium">{{ value }}</OuiText>
                    </template>
                    <template #cell-dataType="{ value }">
                      <OuiText color="tertiary" style="font-family: monospace">{{ value }}</OuiText>
                    </template>
                    <template #cell-isNullable="{ value }">
                      <OuiBadge :color="value ? 'secondary' : 'warning'" size="xs">
                        {{ value ? 'Yes' : 'No' }}
                      </OuiBadge>
                    </template>
                    <template #cell-defaultValue="{ value }">
                      <OuiText color="tertiary">{{ value || '—' }}</OuiText>
                    </template>
                    <template #cell-isPrimaryKey="{ value }">
                      <OuiBadge v-if="value" color="primary" size="xs">PK</OuiBadge>
                      <span v-else style="color: var(--oui-text-secondary)">—</span>
                    </template>
                    <template #cell-isUnique="{ value }">
                      <OuiBadge v-if="value" color="info" size="xs">Unique</OuiBadge>
                      <span v-else style="color: var(--oui-text-secondary)">—</span>
                    </template>
                    <template #cell-actions="{ row }">
                      <OuiButton
                        v-if="!row.isPrimaryKey"
                        variant="ghost"
                        size="sm"
                        color="danger"
                        @click="dropColumn(row.name)"
                        title="Drop column"
                      >
                        <TrashIcon style="width: 0.875rem; height: 0.875rem" />
                      </OuiButton>
                    </template>
                  </OuiTable>
                </div>
              </div>
            </template>

            <!-- Indexes tab -->
            <template #indexes>
              <div style="flex: 1; overflow: hidden; display: flex; flex-direction: column">
                <OuiFlex align="center" gap="sm" style="padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--oui-border-default); background: var(--oui-surface-base)">
                  <OuiButton variant="ghost" color="primary" size="sm" @click="showCreateIndex = true">
                    <PlusIcon style="width: 0.875rem; height: 0.875rem" />
                    Create Index
                  </OuiButton>
                </OuiFlex>
                <div style="flex: 1; overflow: auto; padding: 1rem">
                  <OuiTable
                    v-if="selectedTable && selectedTable.indexes.length > 0"
                    :columns="indexColumns"
                    :rows="selectedTable.indexes"
                    row-key="name"
                    empty-text="No indexes"
                  >
                    <template #cell-name="{ value }">
                      <OuiText weight="medium">{{ value }}</OuiText>
                    </template>
                    <template #cell-columnNames="{ value }">
                      <OuiText color="tertiary" style="font-family: monospace">{{ value.join(', ') }}</OuiText>
                    </template>
                    <template #cell-type="{ value }">
                      <OuiText color="tertiary">{{ value || '—' }}</OuiText>
                    </template>
                    <template #cell-isUnique="{ value }">
                      <OuiBadge v-if="value" color="info" size="xs">Yes</OuiBadge>
                      <span v-else style="color: var(--oui-text-secondary)">No</span>
                    </template>
                    <template #cell-isPrimary="{ value }">
                      <OuiBadge v-if="value" color="primary" size="xs">Yes</OuiBadge>
                      <span v-else style="color: var(--oui-text-secondary)">No</span>
                    </template>
                    <template #cell-actions="{ row }">
                      <OuiButton
                        v-if="!row.isPrimary"
                        variant="ghost"
                        size="sm"
                        color="danger"
                        @click="dropIndex(row.name)"
                        title="Drop index"
                      >
                        <TrashIcon style="width: 0.875rem; height: 0.875rem" />
                      </OuiButton>
                    </template>
                  </OuiTable>
                  <OuiText v-else color="tertiary" size="sm" style="padding: 1rem; text-align: center">
                    No indexes found
                  </OuiText>
                </div>
              </div>
            </template>

            <!-- Foreign Keys tab -->
            <template #foreignKeys>
              <div style="flex: 1; overflow: auto; padding: 1rem">
                <OuiTable
                  v-if="selectedTable && selectedTable.foreignKeys.length > 0"
                  :columns="fkColumns"
                  :rows="selectedTable.foreignKeys"
                  row-key="name"
                  empty-text="No foreign keys"
                >
                  <template #cell-name="{ value }">
                    <OuiText weight="medium">{{ value }}</OuiText>
                  </template>
                  <template #cell-fromColumns="{ value }">
                    <OuiText color="tertiary" style="font-family: monospace">{{ value.join(', ') }}</OuiText>
                  </template>
                  <template #cell-toTable="{ row }">
                    <OuiText style="font-family: monospace">
                      <span style="color: var(--oui-primary)">{{ row.toTable }}</span>.{{ row.toColumns.join(', ') }}
                    </OuiText>
                  </template>
                  <template #cell-onDelete="{ value }">
                    <OuiText color="tertiary">{{ value || '—' }}</OuiText>
                  </template>
                  <template #cell-onUpdate="{ value }">
                    <OuiText color="tertiary">{{ value || '—' }}</OuiText>
                  </template>
                </OuiTable>
                <OuiText v-else color="tertiary" size="sm" style="padding: 1rem; text-align: center">
                  No foreign keys found
                </OuiText>
              </div>
            </template>

            <!-- DDL tab -->
            <template #ddl>
              <div style="flex: 1; overflow: hidden; display: flex; flex-direction: column">
                <OuiFlex align="center" gap="sm" style="padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--oui-border-default); background: var(--oui-surface-base)">
                  <OuiButton variant="ghost" color="secondary" size="sm" @click="loadTableDDL" :loading="loadingDDL">
                    <ArrowPathIcon style="width: 0.875rem; height: 0.875rem" />
                    Refresh
                  </OuiButton>
                  <OuiButton variant="ghost" color="secondary" size="sm" @click="copyDDL" :disabled="!tableDDL">
                    <ClipboardDocumentIcon style="width: 0.875rem; height: 0.875rem" />
                    Copy
                  </OuiButton>
                </OuiFlex>
                <div style="flex: 1; overflow: auto; padding: 1rem">
                  <OuiStack v-if="loadingDDL" align="center" gap="sm" style="padding: 2rem">
                    <OuiSpinner />
                    <OuiText color="tertiary" size="xs">Loading DDL...</OuiText>
                  </OuiStack>
                  <pre
                    v-else-if="tableDDL"
                    style="font-size: 0.75rem; font-family: monospace; background: var(--oui-surface-base); border: 1px solid var(--oui-border-default); border-radius: 0.5rem; padding: 1rem; overflow-x: auto; white-space: pre-wrap"
                  >{{ tableDDL }}</pre>
                  <OuiText v-else color="tertiary" size="sm" style="padding: 1rem; text-align: center">
                    No DDL available
                  </OuiText>
                </div>
              </div>
            </template>
          </OuiTabs>
        </template>

        <!-- No table selected -->
        <OuiStack v-else align="center" justify="center" style="flex: 1; padding: 4rem 0">
          <TableCellsIcon style="width: 3rem; height: 3rem; color: var(--oui-text-muted)" />
          <OuiText color="tertiary" size="sm">Select a table to browse</OuiText>
        </OuiStack>
      </div>
    </div>

    <!-- Add Column Dialog -->
    <OuiDialog v-model:open="showAddColumn" title="Add Column" size="sm">
      <OuiStack gap="md">
        <OuiFormField label="Column Name" required>
          <OuiInput v-model="newColumn.name" placeholder="column_name" />
        </OuiFormField>
        <OuiFormField label="Data Type" required>
          <OuiSelect
            v-model="newColumn.dataType"
            :items="columnTypeOptions"
            placeholder="Select type"
          />
        </OuiFormField>
        <OuiFlex gap="md">
          <OuiCheckbox v-model="newColumn.isNullable">Nullable</OuiCheckbox>
          <OuiCheckbox v-model="newColumn.isUnique">Unique</OuiCheckbox>
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
          <div style="max-height: 160px; overflow-y: auto; border: 1px solid var(--oui-border-default); border-radius: 0.375rem; padding: 0.5rem">
            <OuiCheckbox
              v-for="col in selectedTable?.columns || []"
              :key="col.name"
              :model-value="newIndex.columnNames.includes(col.name)"
              @update:model-value="toggleIndexColumn(col.name)"
              style="display: block; padding: 0.25rem 0"
            >
              {{ col.name }} <span style="color: var(--oui-text-secondary); font-size: 0.75rem; margin-left: 0.5rem">{{ col.dataType }}</span>
            </OuiCheckbox>
          </div>
        </OuiFormField>
        <OuiCheckbox v-model="newIndex.isUnique">Unique Index</OuiCheckbox>
      </OuiStack>
      <template #footer>
        <OuiButton variant="ghost" @click="showCreateIndex = false">Cancel</OuiButton>
        <OuiButton color="primary" @click="createIndex" :disabled="!newIndex.name || newIndex.columnNames.length === 0">
          Create Index
        </OuiButton>
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

    <!-- Cell Detail Dialog -->
    <OuiDialog v-model:open="cellDetailOpen" title="Cell Value">
      <OuiStack v-if="cellDetail" gap="md">
        <OuiFlex gap="sm" align="center" wrap="wrap">
          <OuiBadge color="primary" size="sm">{{ cellDetail.column.name }}</OuiBadge>
          <OuiText size="xs" color="tertiary" class="font-mono">{{ cellDetail.column.dataType }}</OuiText>
          <OuiBadge v-if="cellDetail.value === null" color="tertiary" size="xs">NULL</OuiBadge>
        </OuiFlex>
        <pre class="db-cell-detail">{{ formatCellValue(cellDetail.value) }}</pre>
      </OuiStack>
      <template #footer>
        <OuiButton variant="ghost" @click="cellDetailOpen = false">Close</OuiButton>
        <OuiButton color="primary" @click="copyCellValue" :disabled="!cellDetail">
          <ClipboardDocumentIcon class="h-3.5 w-3.5" />
          Copy value
        </OuiButton>
      </template>
    </OuiDialog>
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
import { ref, computed, onMounted, onUnmounted, toRef, watch, nextTick } from "vue";
import { DatabaseService, type QueryResultRow } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useToast } from "~/composables/useToast";
import { useDialog } from "~/composables/useDialog";
import { useDatabaseSchema, type SchemaTable, type SchemaColumn, type SchemaForeignKey } from "~/composables/useDatabaseSchema";
import ErrorAlert from "~/components/ErrorAlert.vue";
import type { TableColumn } from "~/components/oui/Table.vue";

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
const contextMenuTable = ref<SchemaTable | null>(null);
const contextMenuOpen = ref(false);
const contextMenuTriggerRef = ref<HTMLElement | null>(null);

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

// Context menu
function openContextMenu(e: MouseEvent, table: SchemaTable) {
  e.preventDefault();
  e.stopPropagation();

  contextMenuTable.value = table;

  // Position the hidden trigger at the click location (matching TreeNode pattern)
  if (contextMenuTriggerRef.value) {
    contextMenuTriggerRef.value.style.position = 'fixed';
    contextMenuTriggerRef.value.style.left = `${e.clientX}px`;
    contextMenuTriggerRef.value.style.top = `${e.clientY}px`;
    contextMenuTriggerRef.value.style.width = '1px';
    contextMenuTriggerRef.value.style.height = '1px';
    contextMenuTriggerRef.value.style.opacity = '0';
    contextMenuTriggerRef.value.style.pointerEvents = 'none';
    contextMenuTriggerRef.value.style.zIndex = '-1';
  }

  // Open menu on next tick to ensure DOM is updated
  nextTick(() => {
    contextMenuOpen.value = true;
  });
}

function handleTableAction(action: string) {
  const table = contextMenuTable.value;
  if (!table) return;

  // Close the menu
  contextMenuOpen.value = false;

  switch (action) {
    case "data":
      selectTable(table);
      activeDataTab.value = "data";
      break;
    case "structure":
      selectTable(table);
      activeDataTab.value = "structure";
      break;
    case "ddl":
      selectTable(table);
      activeDataTab.value = "ddl";
      break;
    case "rename":
      openRenameDialog(table);
      break;
    case "truncate":
      truncateSelectedTable(table);
      break;
    case "drop":
      dropSelectedTable(table);
      break;
  }
}

// Tabs
const dataTabs = [
  { id: "data", label: "Data" },
  { id: "structure", label: "Structure" },
  { id: "indexes", label: "Indexes" },
  { id: "foreignKeys", label: "Foreign Keys" },
  { id: "ddl", label: "DDL" },
];
const activeDataTab = ref("data");

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

// Column type options for OuiSelect
const columnTypeOptions = [
  { label: "varchar(255)", value: "varchar(255)" },
  { label: "text", value: "text" },
  { label: "char(1)", value: "char(1)" },
  { label: "uuid", value: "uuid" },
  { label: "integer", value: "integer" },
  { label: "bigint", value: "bigint" },
  { label: "smallint", value: "smallint" },
  { label: "decimal", value: "decimal" },
  { label: "numeric", value: "numeric" },
  { label: "real", value: "real" },
  { label: "double precision", value: "double precision" },
  { label: "timestamp", value: "timestamp" },
  { label: "timestamptz", value: "timestamptz" },
  { label: "date", value: "date" },
  { label: "time", value: "time" },
  { label: "boolean", value: "boolean" },
  { label: "jsonb", value: "jsonb" },
  { label: "json", value: "json" },
  { label: "bytea", value: "bytea" },
];

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
const dataFilter = ref("");

// Per page options for OuiSelect
const perPageOptions = [
  { label: "25", value: 25 },
  { label: "50", value: 50 },
  { label: "100", value: 100 },
  { label: "200", value: 200 },
];

// Inline editing
const editingCell = ref<{ row: number; col: string; value: string } | null>(null);
const pendingEdits = ref(new Map<string, { rowIdx: number; col: string; oldValue: any; newValue: string }>());
const savingEdits = ref(false);

// Insert row
const insertingRow = ref(false);
const newRowValues = ref<Record<string, string>>({});
const cellDetailOpen = ref(false);
const cellDetail = ref<{ column: { name: string; dataType: string }; value: any } | null>(null);

const filteredTables = computed(() => {
  if (!searchQuery.value) return schemaTables.value;
  const q = searchQuery.value.toLowerCase();
  return schemaTables.value.filter((t: SchemaTable) => t.name.toLowerCase().includes(q));
});

const selectedTable = computed(() => {
  if (!selectedTableName.value) return null;
  return schemaTables.value.find((t: SchemaTable) => t.name === selectedTableName.value) || null;
});

const primaryKeyColumns = computed(() => {
  return selectedTable.value?.columns
    .filter((column: SchemaColumn) => column.isPrimaryKey)
    .map((column: SchemaColumn) => column.name) || [];
});

const filteredDataRows = computed(() => {
  const q = dataFilter.value.trim().toLowerCase();
  if (!q) return dataRows.value;

  return dataRows.value.filter((row) => {
    return Object.entries(row).some(([key, value]) => {
      if (key.startsWith("__")) return false;
      return formatCellValue(value).toLowerCase().includes(q);
    });
  });
});

// Table columns for OuiTable
const tableColumns = computed<TableColumn[]>(() => {
  if (!dataResponse.value) return [];
  const cols: TableColumn[] = [
    { key: "__rowNum", label: "#", width: 50, sortable: false, resizable: false },
  ];
  for (const col of dataResponse.value.columns || []) {
    cols.push({
      key: col.name,
      label: col.name,
      minWidth: getColumnMinWidth(col.name, col.dataType),
      defaultWidth: getColumnWidth(col.name, col.dataType),
      sortable: true,
    });
  }
  cols.push({ key: "__actions", label: "", width: 50, sortable: false, resizable: false });
  return cols;
});

// Structure table columns
const structureColumns: TableColumn[] = [
  { key: "name", label: "Column", minWidth: 120 },
  { key: "dataType", label: "Type", minWidth: 100 },
  { key: "isNullable", label: "Nullable", width: 80 },
  { key: "defaultValue", label: "Default", minWidth: 100 },
  { key: "isPrimaryKey", label: "PK", width: 60 },
  { key: "isUnique", label: "Unique", width: 80 },
  { key: "actions", label: "", width: 60 },
];

// Index table columns
const indexColumns: TableColumn[] = [
  { key: "name", label: "Name", minWidth: 150 },
  { key: "columnNames", label: "Columns", minWidth: 150 },
  { key: "type", label: "Type", width: 100 },
  { key: "isUnique", label: "Unique", width: 80 },
  { key: "isPrimary", label: "Primary", width: 80 },
  { key: "actions", label: "", width: 60 },
];

// FK table columns
const fkColumns: TableColumn[] = [
  { key: "name", label: "Name", minWidth: 150 },
  { key: "fromColumns", label: "From", minWidth: 120 },
  { key: "toTable", label: "To", minWidth: 180 },
  { key: "onDelete", label: "On Delete", width: 100 },
  { key: "onUpdate", label: "On Update", width: 100 },
];

function toggleTableExpand(name: string) {
  if (expandedTables.value.has(name)) {
    expandedTables.value.delete(name);
  } else {
    expandedTables.value.add(name);
  }
}

function isForeignKey(table: SchemaTable, colName: string): boolean {
  return table.foreignKeys.some((fk: SchemaForeignKey) => fk.fromColumns.includes(colName));
}

function getColumnWidth(name: string, dataType: string): number {
  const normalizedType = dataType.toLowerCase();
  const normalizedName = name.toLowerCase();

  if (normalizedName === "id" || normalizedName.endsWith("_id") || normalizedType.includes("uuid")) {
    return 220;
  }
  if (normalizedType.includes("json") || normalizedType.includes("text")) {
    return 320;
  }
  if (normalizedType.includes("timestamp") || normalizedType.includes("date") || normalizedType.includes("time")) {
    return 180;
  }
  if (normalizedType.includes("bool")) {
    return 110;
  }
  if (
    normalizedType.includes("int") ||
    normalizedType.includes("numeric") ||
    normalizedType.includes("decimal") ||
    normalizedType.includes("float") ||
    normalizedType.includes("double")
  ) {
    return 120;
  }
  return 180;
}

function getColumnMinWidth(name: string, dataType: string): number {
  const normalizedType = dataType.toLowerCase();
  const normalizedName = name.toLowerCase();

  if (normalizedName === "id") {
    return 72;
  }
  if (normalizedName.endsWith("_id") || normalizedType.includes("uuid")) {
    return 96;
  }
  if (
    normalizedType.includes("int") ||
    normalizedType.includes("numeric") ||
    normalizedType.includes("decimal") ||
    normalizedType.includes("float") ||
    normalizedType.includes("double") ||
    normalizedType.includes("bool")
  ) {
    return 72;
  }
  return 96;
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

// Handle OuiTable sort
function handleTableSort(column: TableColumn, direction: "asc" | "desc" | null) {
  if (direction === null) {
    dataSortColumn.value = null;
  } else {
    dataSortColumn.value = column.key;
    dataSortDirection.value = direction.toUpperCase() as "ASC" | "DESC";
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
    dataRows.value = (res.rows || []).map((row: QueryResultRow, idx: number) => {
      const obj: Record<string, any> = { __rowIdx: idx };
      for (const cell of row.cells || []) {
        obj[cell.columnName] = cell.isNull ? null : cell.value;
      }
      return obj;
    });
  } catch (err: unknown) {
    toast.error("Failed to load table data", (err as Error).message);
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
}

function confirmCellEdit() {
  if (!editingCell.value) return;

  const { row, col, value } = editingCell.value;
  const rowData = dataRows.value[row];
  if (!rowData) return;
  const oldValue = rowData[col];
  const newValue = value;

  if (String(oldValue ?? "") !== newValue) {
    const key = `${row}:${col}`;
    pendingEdits.value.set(key, { rowIdx: row, col, oldValue, newValue });
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

function isBooleanValue(value: any): boolean {
  if (typeof value === "boolean") return true;
  if (typeof value !== "string") return false;
  const normalized = value.toLowerCase();
  return normalized === "true" || normalized === "false" || normalized === "t" || normalized === "f";
}

function isTruthyBoolean(value: any): boolean {
  if (typeof value === "boolean") return value;
  const normalized = String(value).toLowerCase();
  return normalized === "true" || normalized === "t" || normalized === "1";
}

function isJsonColumn(dataType: string): boolean {
  const normalized = dataType.toLowerCase();
  return normalized.includes("json");
}

function formatJsonPreview(value: any): string {
  const normalized = normalizeCellValue(value);
  if (normalized === null || normalized === undefined) return "NULL";
  try {
    const parsed = typeof normalized === "string" ? JSON.parse(normalized) : normalized;
    return JSON.stringify(parsed);
  } catch {
    return String(normalized);
  }
}

function formatCellValue(value: any): string {
  const normalized = normalizeCellValue(value);
  if (normalized === null || normalized === undefined) return "NULL";
  if (typeof normalized === "object") {
    try {
      return JSON.stringify(normalized, null, 2);
    } catch {
      return String(normalized);
    }
  }
  const stringValue = String(normalized);
  try {
    const parsed = JSON.parse(stringValue);
    if (parsed && typeof parsed === "object") {
      return JSON.stringify(parsed, null, 2);
    }
  } catch {
    // Plain strings are expected most of the time.
  }
  return stringValue;
}

function normalizeCellValue(value: any): any {
  if (Array.isArray(value) && value.every((item) => Number.isInteger(item) && item >= 0 && item <= 255)) {
    return String.fromCharCode(...value);
  }
  if (value instanceof Uint8Array) {
    return new TextDecoder().decode(value);
  }
  if (typeof value === "string") {
    const bytes = parseByteArrayString(value);
    if (bytes) {
      return String.fromCharCode(...bytes);
    }
  }
  return value;
}

function parseByteArrayString(value: string): number[] | null {
  const trimmed = value.trim();
  if (!/^\[(\d+\s*)+\]$/.test(trimmed)) {
    return null;
  }

  const bytes = trimmed
    .slice(1, -1)
    .trim()
    .split(/\s+/)
    .map((part) => Number(part));

  if (!bytes.length || bytes.some((byte) => !Number.isInteger(byte) || byte < 0 || byte > 255)) {
    return null;
  }

  return bytes;
}

function openCellDetail(row: Record<string, any>, column: { name: string; dataType: string }) {
  cellDetail.value = {
    column,
    value: row[column.name],
  };
  cellDetailOpen.value = true;
}

async function copyCellValue() {
  if (!cellDetail.value) return;
  try {
    await navigator.clipboard.writeText(formatCellValue(cellDetail.value.value));
    toast.success("Cell value copied");
  } catch {
    toast.error("Failed to copy cell value");
  }
}

async function copyTableName() {
  if (!selectedTableName.value) return;
  try {
    await navigator.clipboard.writeText(selectedTableName.value);
    toast.success("Table name copied");
  } catch {
    toast.error("Failed to copy table name");
  }
}

function discardEdits() {
  pendingEdits.value.clear();
  editingCell.value = null;
  loadTableData();
}

// Save edits
async function saveEdits() {
  if (!selectedTableName.value || !organizationId.value || !selectedTable.value) return;

  savingEdits.value = true;
  const pkColumns = selectedTable.value.columns.filter((c: SchemaColumn) => c.isPrimaryKey);

  try {
    for (const edit of pendingEdits.value.values()) {
      const row = dataRows.value[edit.rowIdx];
      if (!row) continue;

      const whereCells = pkColumns.map((pk: SchemaColumn) => ({
        columnName: pk.name,
        value: row[pk.name] !== null ? String(row[pk.name]) : undefined,
        isNull: row[pk.name] === null,
      }));

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
  } catch (err: unknown) {
    toast.error("Failed to save changes", (err as Error).message);
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
  } catch (err: unknown) {
    toast.error("Failed to insert row", (err as Error).message);
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

  const pkColumns = selectedTable.value.columns.filter((c: SchemaColumn) => c.isPrimaryKey);
  if (pkColumns.length === 0) {
    toast.error("Cannot delete: table has no primary key");
    return;
  }

  const row = dataRows.value[rowIdx];
  if (!row) return;
  const whereCells = pkColumns.map((pk: SchemaColumn) => ({
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
  } catch (err: unknown) {
    toast.error("Failed to delete row", (err as Error).message);
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
  } catch (err: unknown) {
    toast.error("Failed to load DDL", (err as Error).message);
    tableDDL.value = "";
  } finally {
    loadingDDL.value = false;
  }
}

watch(activeDataTab, (tab) => {
  if (tab === "ddl" && selectedTableName.value) {
    loadTableDDL();
  }
});

async function copyDDL() {
  if (!tableDDL.value) return;
  try {
    await navigator.clipboard.writeText(tableDDL.value);
    toast.success("DDL copied to clipboard");
  } catch {
    toast.error("Failed to copy to clipboard");
  }
}

function openCreateTableDialog() {
  showCreateTable.value = true;
}

function openRenameDialog(table: SchemaTable) {
  selectedTableName.value = table.name;
  renameTableName.value = table.name;
  showRenameTable.value = true;
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
  } catch (err: unknown) {
    toast.error("Failed to add column", (err as Error).message);
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
  } catch (err: unknown) {
    toast.error("Failed to drop column", (err as Error).message);
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
  } catch (err: unknown) {
    toast.error("Failed to create index", (err as Error).message);
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
  } catch (err: unknown) {
    toast.error("Failed to drop index", (err as Error).message);
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
  } catch (err: unknown) {
    toast.error("Failed to rename table", (err as Error).message);
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
  } catch (err: unknown) {
    toast.error("Failed to truncate table", (err as Error).message);
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
  } catch (err: unknown) {
    toast.error("Failed to drop table", (err as Error).message);
  }
}

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
});

onUnmounted(() => {
  document.removeEventListener("mousemove", onTreeResize);
  document.removeEventListener("mouseup", stopTreeResize);
});
</script>

<style scoped>
.db-browser-shell {
  display: flex;
  min-height: 500px;
  height: min(68vh, 760px);
}

.db-schema-pane {
  display: flex;
  min-height: 0;
  flex-shrink: 0;
  overflow: hidden;
  border-right: 1px solid var(--oui-border-default);
  background: var(--oui-surface-base);
}

.db-schema-content {
  display: flex;
  min-height: 0;
  width: 100%;
  flex-direction: column;
  padding: 0.75rem;
}

.db-schema-collapsible {
  min-height: 0;
}

.db-schema-list {
  display: flex;
  min-height: 0;
  max-height: calc(68vh - 12rem);
  flex-direction: column;
  gap: 0.125rem;
  overflow-y: auto;
  padding: 0.125rem 0.125rem 0.125rem 0.5rem;
}

.db-schema-table {
  flex: 0 0 auto;
}

.db-schema-table-row {
  display: flex;
  width: 100%;
  min-height: 1.625rem;
  align-items: center;
  gap: 0.375rem;
  border: 0;
  border-radius: 0.375rem;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  text-align: left;
  font-size: 0.75rem;
  line-height: 1rem;
  transition: color 0.15s ease, background-color 0.15s ease;
}

.db-schema-collapsible :deep([data-part="content"]) {
  min-height: 0;
  overflow: hidden;
}

.db-workbench-header {
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--oui-border-default);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--oui-surface-base) 82%, transparent), var(--oui-surface-base)),
    var(--oui-surface-base);
}

.db-grid-toolbar {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--oui-border-default);
  background: var(--oui-surface-base);
}

.db-grid-search {
  width: min(24rem, 100%);
}

.db-grid-shell {
  flex: 1;
  min-height: 0;
  overflow: auto;
  background: var(--oui-surface-overlay);
}

.db-insert-row {
  padding: 0.875rem 1rem;
  border-bottom: 1px solid var(--oui-border-default);
  background: color-mix(in srgb, var(--oui-accent-success) 8%, var(--oui-surface-base));
}

.db-column-heading {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.125rem;
  line-height: 1.15;
}

.db-column-type {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.625rem;
  font-weight: 500;
  color: var(--oui-text-tertiary);
  text-transform: none;
}

.db-cell-input {
  width: 100%;
  border: 0;
  border-bottom: 1px solid var(--oui-accent-primary);
  outline: none;
  background: transparent;
  padding: 0.25rem 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.75rem;
}

.db-cell-value {
  display: flex;
  width: 100%;
  min-width: 0;
  max-width: 100%;
  align-items: center;
  border: 0;
  border-radius: 0.375rem;
  background: transparent;
  color: inherit;
  cursor: zoom-in;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.75rem;
  line-height: 1.35;
  overflow: hidden;
  padding: 0.25rem 0.375rem;
  text-align: left;
}

.db-cell-value > span {
  min-width: 0;
}

.db-cell-value:hover {
  background: var(--oui-surface-hover);
}

.db-cell-edited {
  background: color-mix(in srgb, var(--oui-accent-warning) 12%, transparent);
  box-shadow: inset 2px 0 0 var(--oui-accent-warning);
}

.db-null {
  color: var(--oui-text-tertiary);
  font-style: italic;
}

.db-json {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--oui-accent-info);
}

.db-boolean {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
}

.db-boolean-dot {
  width: 0.45rem;
  height: 0.45rem;
  border-radius: 999px;
  background: var(--oui-text-muted);
}

.db-boolean-dot.is-true {
  background: var(--oui-accent-success);
}

.db-boolean-dot.is-false {
  background: var(--oui-text-muted);
}

.db-cell-detail {
  max-height: 24rem;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  border: 1px solid var(--oui-border-default);
  border-radius: 0.5rem;
  background: var(--oui-surface-base);
  padding: 1rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.8125rem;
  line-height: 1.6;
}

:deep(.db-table-wrapper) {
  height: 100%;
  margin-left: 0;
  margin-right: 0;
}

:deep(.db-data-table) {
  width: max-content !important;
  min-width: 100%;
  table-layout: fixed !important;
}

:deep(.db-data-table thead) {
  position: sticky;
  top: 0;
  z-index: 5;
}

:deep(.db-table-header) {
  background: var(--oui-surface-base);
  color: var(--oui-text-secondary);
  letter-spacing: 0;
  text-transform: none;
  border-bottom: 1px solid var(--oui-border-default);
  vertical-align: bottom;
}

:deep(.db-table-cell) {
  border-right: 1px solid color-mix(in srgb, var(--oui-border-default) 60%, transparent);
  vertical-align: middle;
}

:deep(.db-data-table tbody tr:hover) {
  background: color-mix(in srgb, var(--oui-accent-primary) 4%, transparent);
}

:deep(.db-data-table th:first-child),
:deep(.db-data-table td:first-child) {
  position: sticky;
  left: 0;
  z-index: 4;
  background: var(--oui-surface-base);
  color: var(--oui-text-tertiary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

:deep(.db-data-table thead th:first-child) {
  z-index: 6;
}

:deep(.db-data-table th:last-child),
:deep(.db-data-table td:last-child) {
  position: sticky;
  right: 0;
  z-index: 4;
  background: var(--oui-surface-base);
  border-left: 1px solid var(--oui-border-default);
  width: 56px !important;
  min-width: 56px !important;
  max-width: 56px !important;
}

:deep(.db-data-table thead th:last-child) {
  z-index: 6;
}
</style>
