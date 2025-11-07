import { getInitials, formatDate, formatDateOnly, formatBytes, formatCurrency } from "~/utils/common";

export const useUtils = () => {
  return {
    getInitials,
    formatDate,
    formatDateOnly,
    formatBytes,
    formatCurrency,
  };
};

