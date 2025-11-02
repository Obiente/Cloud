<template>
  <DatePicker.Root
    :selection-mode="selectionMode"
    :model-value="modelValue"
    @update:model-value="handleValueChange"
    :min="min"
    :max="max"
    :placeholder="placeholder"
    :disabled="disabled"
    :time-zone="timeZone"
    :locale="locale"
    class="oui-date-picker space-y-1 w-full"
  >
    <DatePicker.Label v-if="label" class="block text-sm font-medium text-primary">
      {{ label }}
    </DatePicker.Label>
    <DatePicker.Control class="relative flex items-center gap-2">
      <DatePicker.Input
        ref="input0Ref"
        :index="0"
        :placeholder="selectionMode === 'range' ? (startPlaceholder || 'Start date') : (placeholder || 'Select date')"
        class="oui-input oui-input-md flex-1 date-input-formatted"
        data-input-index="0"
      />
      <DatePicker.Input
        v-if="selectionMode === 'range'"
        ref="input1Ref"
        :index="1"
        :placeholder="endPlaceholder || 'End date'"
        class="oui-input oui-input-md flex-1 date-input-formatted"
        data-input-index="1"
      />
      <DatePicker.Trigger
        class="flex items-center justify-center h-10 w-10 rounded-xl border border-border-default bg-surface-base text-text-secondary hover:text-primary hover:bg-surface-raised transition-colors duration-150 disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:bg-surface-base"
      >
        <CalendarIcon class="h-4 w-4" />
      </DatePicker.Trigger>
      <DatePicker.ClearTrigger
        v-if="clearable"
        class="flex items-center justify-center h-10 w-10 rounded-xl border border-border-default bg-surface-base text-text-secondary hover:text-primary hover:bg-surface-raised transition-colors duration-150 disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:bg-surface-base"
      >
        <XMarkIcon class="h-4 w-4" />
      </DatePicker.ClearTrigger>
    </DatePicker.Control>

    <Teleport to="body">
      <DatePicker.Positioner>
        <DatePicker.Content
          class="z-50 min-w-[20rem] rounded-xl border border-border-default bg-surface-base shadow-md p-4 animate-in fade-in-0 zoom-in-95 duration-200 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95"
        >
          <div class="flex gap-2 mb-4">
            <DatePicker.YearSelect class="flex-1 oui-input oui-input-sm" />
            <DatePicker.MonthSelect class="flex-1 oui-input oui-input-sm" />
          </div>
          <DatePicker.View view="day">
            <DatePicker.Context v-slot="api">
              <DatePicker.ViewControl
                class="flex items-center justify-between mb-4"
              >
                <DatePicker.PrevTrigger
                  class="flex items-center justify-center h-8 w-8 rounded-md hover:bg-surface-raised text-primary transition-colors duration-150 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  <ChevronLeftIcon class="h-4 w-4" />
                </DatePicker.PrevTrigger>
                <DatePicker.ViewTrigger class="px-4 py-2 font-medium text-primary hover:bg-surface-raised rounded-md transition-colors duration-150">
                  <DatePicker.RangeText />
                </DatePicker.ViewTrigger>
                <DatePicker.NextTrigger
                  class="flex items-center justify-center h-8 w-8 rounded-md hover:bg-surface-raised text-primary transition-colors duration-150 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  <ChevronRightIcon class="h-4 w-4" />
                </DatePicker.NextTrigger>
              </DatePicker.ViewControl>
              <DatePicker.Table>
                <DatePicker.TableHead>
                  <DatePicker.TableRow>
                    <DatePicker.TableHeader
                      v-for="(weekDay, id) in api.weekDays"
                      :key="id"
                      class="p-2 text-xs font-semibold text-text-tertiary uppercase tracking-wide"
                    >
                      {{ weekDay.short }}
                    </DatePicker.TableHeader>
                  </DatePicker.TableRow>
                </DatePicker.TableHead>
                <DatePicker.TableBody>
                  <DatePicker.TableRow
                    v-for="(week, id) in api.weeks"
                    :key="id"
                  >
                    <DatePicker.TableCell
                      v-for="(day, dayId) in week"
                      :key="dayId"
                      :value="day"
                      class="p-1 relative oui-date-picker-cell"
                    >
                      <DatePicker.TableCellTrigger
                        class="relative flex items-center justify-center w-8 h-8 rounded-md text-sm font-medium text-primary hover:bg-surface-raised transition-colors duration-150 data-selected:bg-primary data-selected:text-white data-selected:z-10 data-in-range:bg-surface-raised/40 data-in-range:hover:bg-surface-raised/60 data-focused:ring-2 data-focused:ring-primary data-focused:ring-offset-1 data-disabled:cursor-not-allowed data-disabled:opacity-40 data-disabled:hover:bg-transparent"
                      >
                        {{ day.day }}
                      </DatePicker.TableCellTrigger>
                    </DatePicker.TableCell>
                  </DatePicker.TableRow>
                </DatePicker.TableBody>
              </DatePicker.Table>
            </DatePicker.Context>
          </DatePicker.View>
          <DatePicker.View view="month">
            <DatePicker.Context v-slot="api">
              <DatePicker.ViewControl
                class="flex items-center justify-between mb-4"
              >
                <DatePicker.PrevTrigger
                  class="flex items-center justify-center h-8 w-8 rounded-md hover:bg-surface-raised text-primary transition-colors duration-150 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  <ChevronLeftIcon class="h-4 w-4" />
                </DatePicker.PrevTrigger>
                <DatePicker.ViewTrigger class="px-4 py-2 font-medium text-primary hover:bg-surface-raised rounded-md transition-colors duration-150">
                  <DatePicker.RangeText />
                </DatePicker.ViewTrigger>
                <DatePicker.NextTrigger
                  class="flex items-center justify-center h-8 w-8 rounded-md hover:bg-surface-raised text-primary transition-colors duration-150 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  <ChevronRightIcon class="h-4 w-4" />
                </DatePicker.NextTrigger>
              </DatePicker.ViewControl>
              <DatePicker.Table>
                <DatePicker.TableBody>
                  <DatePicker.TableRow
                    v-for="(months, id) in api.getMonthsGrid({
                      columns: 4,
                      format: 'short',
                    })"
                    :key="id"
                  >
                    <DatePicker.TableCell
                      v-for="(month, monthId) in months"
                      :key="monthId"
                      :value="month.value"
                      class="p-1"
                    >
                      <DatePicker.TableCellTrigger
                        class="px-4 py-2 rounded-md text-sm font-medium text-primary hover:bg-surface-raised transition-colors duration-150 data-selected:bg-primary data-selected:text-white data-focused:ring-2 data-focused:ring-primary data-focused:ring-offset-1 data-disabled:cursor-not-allowed data-disabled:opacity-40"
                      >
                        {{ month.label }}
                      </DatePicker.TableCellTrigger>
                    </DatePicker.TableCell>
                  </DatePicker.TableRow>
                </DatePicker.TableBody>
              </DatePicker.Table>
            </DatePicker.Context>
          </DatePicker.View>
          <DatePicker.View view="year">
            <DatePicker.Context v-slot="api">
              <DatePicker.ViewControl
                class="flex items-center justify-between mb-4"
              >
                <DatePicker.PrevTrigger
                  class="flex items-center justify-center h-8 w-8 rounded-md hover:bg-surface-raised text-primary transition-colors duration-150 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  <ChevronLeftIcon class="h-4 w-4" />
                </DatePicker.PrevTrigger>
                <DatePicker.ViewTrigger class="px-4 py-2 font-medium text-primary hover:bg-surface-raised rounded-md transition-colors duration-150">
                  <DatePicker.RangeText />
                </DatePicker.ViewTrigger>
                <DatePicker.NextTrigger
                  class="flex items-center justify-center h-8 w-8 rounded-md hover:bg-surface-raised text-primary transition-colors duration-150 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  <ChevronRightIcon class="h-4 w-4" />
                </DatePicker.NextTrigger>
              </DatePicker.ViewControl>
              <DatePicker.Table>
                <DatePicker.TableBody>
                  <DatePicker.TableRow
                    v-for="(years, id) in api.getYearsGrid({ columns: 4 })"
                    :key="id"
                  >
                    <DatePicker.TableCell
                      v-for="(year, yearId) in years"
                      :key="yearId"
                      :value="year.value"
                      class="p-1"
                    >
                      <DatePicker.TableCellTrigger
                        class="px-4 py-2 rounded-md text-sm font-medium text-primary hover:bg-surface-raised transition-colors duration-150 data-selected:bg-primary data-selected:text-white data-focused:ring-2 data-focused:ring-primary data-focused:ring-offset-1 data-disabled:cursor-not-allowed data-disabled:opacity-40"
                      >
                        {{ year.label }}
                      </DatePicker.TableCellTrigger>
                    </DatePicker.TableCell>
                  </DatePicker.TableRow>
                </DatePicker.TableBody>
              </DatePicker.Table>
            </DatePicker.Context>
          </DatePicker.View>
        </DatePicker.Content>
      </DatePicker.Positioner>
    </Teleport>
  </DatePicker.Root>
</template>

<script setup lang="ts">
  import { DatePicker, type DateValue } from "@ark-ui/vue/date-picker";
  import {
    CalendarIcon,
    XMarkIcon,
    ChevronLeftIcon,
    ChevronRightIcon,
  } from "@heroicons/vue/24/outline";
  import { nextTick, onMounted, onUnmounted, ref } from "vue";

  interface Props {
    label?: string;
    placeholder?: string;
    startPlaceholder?: string;
    endPlaceholder?: string;
    selectionMode?: "single" | "multiple" | "range";
    modelValue?: DateValue[];
    min?: DateValue;
    max?: DateValue;
    disabled?: boolean;
    clearable?: boolean;
    timeZone?: string;
    locale?: string;
  }

  const props = withDefaults(defineProps<Props>(), {
    selectionMode: "single",
    clearable: true,
    timeZone: "UTC",
    locale: "en-US",
  });

  const emit = defineEmits<{
    "update:modelValue": [value: DateValue[] | undefined];
  }>();

  const handleValueChange = (value: DateValue[] | undefined) => {
    emit("update:modelValue", value);
  };

  const input0Ref = ref<any>();
  const input1Ref = ref<any>();

  // Format date input with automatic "/" separators (MM/DD/YYYY)
  const formatDateInput = (event: Event) => {
    const input = event.target as HTMLInputElement;
    if (!input) return;

    let value = input.value;
    
    // Remove all non-digits
    const digits = value.replace(/\D/g, "");
    
    // Don't format if user is deleting
    if (digits.length === 0) {
      return;
    }

    // Format with slashes: MM/DD/YYYY
    let formatted = "";
    if (digits.length >= 1) {
      formatted = digits.substring(0, 2);
    }
    if (digits.length >= 3) {
      formatted += "/" + digits.substring(2, 4);
    }
    if (digits.length >= 5) {
      formatted += "/" + digits.substring(4, 8);
    }

    // Update input value if it changed
    if (formatted !== value) {
      const cursorPosition = formatted.length;
      input.value = formatted;
      
      // Trigger input event for Ark UI
      input.dispatchEvent(new Event('input', { bubbles: true }));
      
      // Check if date is complete (MM/DD/YYYY format - 10 characters)
      if (formatted.length === 10) {
        // Date is complete, trigger blur to commit the value
        nextTick(() => {
          input.blur();
          // Also trigger change event for Ark UI
          input.dispatchEvent(new Event('change', { bubbles: true }));
        });
      } else {
        // Restore cursor position if not complete
        nextTick(() => {
          input.setSelectionRange(cursorPosition, cursorPosition);
        });
      }
    } else {
      // Value didn't change but check if it's complete
      if (formatted.length === 10) {
        // Date is complete, trigger blur to commit
        nextTick(() => {
          input.blur();
          input.dispatchEvent(new Event('change', { bubbles: true }));
        });
      }
    }
  };

  // Handle special keys (backspace, delete) to allow proper deletion
  const handleDateInputKeydown = (event: KeyboardEvent) => {
    const input = event.target as HTMLInputElement;
    if (!input) return;

    // Allow backspace and delete to work normally
    if (event.key === "Backspace" || event.key === "Delete") {
      // If deleting a "/", also delete the preceding digit
      if (input.selectionStart === input.selectionEnd) {
        const pos = input.selectionStart || 0;
        const value = input.value;
        
        if (value[pos - 1] === "/" && event.key === "Backspace") {
          // Delete both the "/" and the preceding digit
          event.preventDefault();
          const newValue = value.substring(0, pos - 2) + value.substring(pos);
          input.value = newValue;
          input.dispatchEvent(new Event('input', { bubbles: true }));
          nextTick(() => {
            input.setSelectionRange(pos - 2, pos - 2);
          });
        }
      }
      return;
    }

    // Allow navigation keys
    if (
      event.key === "ArrowLeft" ||
      event.key === "ArrowRight" ||
      event.key === "ArrowUp" ||
      event.key === "ArrowDown" ||
      event.key === "Home" ||
      event.key === "End" ||
      event.key === "Tab" ||
      event.ctrlKey ||
      event.metaKey
    ) {
      return;
    }

    // Only allow digits
    if (!/^\d$/.test(event.key)) {
      event.preventDefault();
      return;
    }

    // Auto-advance past "/" separators
    const pos = input.selectionStart || 0;
    const value = input.value;
    
    if (value[pos] === "/") {
      // If cursor is on a "/", move past it
      nextTick(() => {
        input.setSelectionRange(pos + 1, pos + 1);
      });
    }
  };

  // Setup event listeners for date input formatting
  onMounted(() => {
    // Use multiple attempts to find inputs since they might not be ready immediately
    const setupInputListeners = () => {
      // Find actual input elements within the DatePicker.Input components
      const findInputElement = (componentRef: any): HTMLInputElement | null => {
        if (!componentRef) return null;
        const el = componentRef.$el || componentRef;
        if (el instanceof HTMLInputElement) return el;
        // Try to find input by querySelector
        return el?.querySelector?.('input') || null;
      };

      // Also try finding by class as fallback
      const inputs = document.querySelectorAll('.date-input-formatted input') as NodeListOf<HTMLInputElement>;
      
      const input0 = findInputElement(input0Ref.value) || (inputs[0] || null);
      const input1 = props.selectionMode === 'range' 
        ? (findInputElement(input1Ref.value) || (inputs[1] || null))
        : null;

      if (input0 && !input0.hasAttribute('data-formatted')) {
        input0.setAttribute('data-formatted', 'true');
        input0.addEventListener('input', formatDateInput, { passive: true });
        input0.addEventListener('keydown', handleDateInputKeydown);
        // Preserve value when blurring to prevent clearing when switching inputs
        input0.addEventListener('blur', (e) => {
          const input = e.target as HTMLInputElement;
          const val = input.value;
          // Preserve value if it has any meaningful content (at least MM/DD format)
          if (val && val.replace(/\D/g, '').length >= 4) {
            // Store for restoration
            const storedValue = val;
            // Use a slight delay to let Ark UI process first, then restore if needed
            setTimeout(() => {
              if (input.value !== storedValue && input.value.length < storedValue.length) {
                input.value = storedValue;
                input.dispatchEvent(new Event('input', { bubbles: true }));
                input.dispatchEvent(new Event('change', { bubbles: true }));
              }
            }, 10);
          }
        }, { capture: true });
      }

      if (input1 && !input1.hasAttribute('data-formatted')) {
        input1.setAttribute('data-formatted', 'true');
        input1.addEventListener('input', formatDateInput, { passive: true });
        input1.addEventListener('keydown', handleDateInputKeydown);
        // Preserve value when blurring to prevent clearing when switching inputs
        input1.addEventListener('blur', (e) => {
          const input = e.target as HTMLInputElement;
          const val = input.value;
          // Preserve value if it has any meaningful content (at least MM/DD format)
          if (val && val.replace(/\D/g, '').length >= 4) {
            // Store for restoration
            const storedValue = val;
            // Use a slight delay to let Ark UI process first, then restore if needed
            setTimeout(() => {
              if (input.value !== storedValue && input.value.length < storedValue.length) {
                input.value = storedValue;
                input.dispatchEvent(new Event('input', { bubbles: true }));
                input.dispatchEvent(new Event('change', { bubbles: true }));
              }
            }, 10);
          }
        }, { capture: true });
      }

      // Return true if at least one input was found
      return !!(input0 || input1);
    };

    // Try immediately
    if (!setupInputListeners()) {
      // If inputs not found, try again after a short delay
      nextTick(() => {
        if (!setupInputListeners()) {
          setTimeout(() => setupInputListeners(), 100);
        }
      });
    }
  });

  onUnmounted(() => {
    const findInputElement = (componentRef: any): HTMLInputElement | null => {
      if (!componentRef) return null;
      const el = componentRef.$el || componentRef;
      if (el instanceof HTMLInputElement) return el;
      return el?.querySelector?.('input') || null;
    };

    const input0 = findInputElement(input0Ref.value);
    const input1 = findInputElement(input1Ref.value);

    if (input0) {
      input0.removeEventListener('input', formatDateInput);
      input0.removeEventListener('keydown', handleDateInputKeydown);
    }

    if (input1) {
      input1.removeEventListener('input', formatDateInput);
      input1.removeEventListener('keydown', handleDateInputKeydown);
    }
  });

  defineOptions({ inheritAttrs: false });
</script>

<style scoped>
/* Range selection styling - create continuous raised surface background */
:deep(.oui-date-picker .oui-date-picker-cell[data-in-range]) {
  background-color: var(--oui-surface-raised);
  opacity: 0.4;
}

:deep(.oui-date-picker .oui-date-picker-cell[data-in-range]::before) {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--oui-surface-raised);
  opacity: 0.4;
  z-index: 0;
}

/* Make the first cell in range have rounded left corners */
:deep(.oui-date-picker .oui-date-picker-cell[data-range-start]) {
  border-top-left-radius: 0.375rem;
  border-bottom-left-radius: 0.375rem;
}

:deep(.oui-date-picker .oui-date-picker-cell[data-range-start]::before) {
  border-top-left-radius: 0.375rem;
  border-bottom-left-radius: 0.375rem;
}

/* Make the last cell in range have rounded right corners */
:deep(.oui-date-picker .oui-date-picker-cell[data-range-end]) {
  border-top-right-radius: 0.375rem;
  border-bottom-right-radius: 0.375rem;
}

:deep(.oui-date-picker .oui-date-picker-cell[data-range-end]::before) {
  border-top-right-radius: 0.375rem;
  border-bottom-right-radius: 0.375rem;
}

/* Ensure selected dates appear above the range background */
:deep(.oui-date-picker [data-selected]) {
  z-index: 10;
  position: relative;
}
</style>
