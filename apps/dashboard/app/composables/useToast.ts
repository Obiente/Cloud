import { createToaster } from "@ark-ui/vue/toast";
import {
  CheckCircleIcon,
  XCircleIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
} from "@heroicons/vue/24/outline";

// Global toaster instance
let globalToaster: ReturnType<typeof createToaster> | null = null;

export function useToast() {
  if (!globalToaster) {
    globalToaster = createToaster({
      placement: "bottom-end",
      overlap: true,
      gap: 16,
    });
  }

  const iconMap = {
    success: CheckCircleIcon,
    error: XCircleIcon,
    warning: ExclamationTriangleIcon,
    info: InformationCircleIcon,
  };

  const toast = {
    create: (options: {
      title: string;
      description?: string;
      type?: "success" | "error" | "warning" | "info";
      duration?: number;
    }) => {
      return globalToaster!.create({
        title: options.title,
        description: options.description,
        type: options.type || "info",
        duration: options.duration || 5000,
      });
    },
    success: (title: string, description?: string) => {
      return globalToaster!.create({
        title,
        description,
        type: "success",
        duration: 5000,
      });
    },
    error: (title: string, description?: string) => {
      return globalToaster!.create({
        title,
        description,
        type: "error",
        duration: 7000,
      });
    },
    warning: (title: string, description?: string) => {
      return globalToaster!.create({
        title,
        description,
        type: "warning",
        duration: 6000,
      });
    },
    info: (title: string, description?: string) => {
      return globalToaster!.create({
        title,
        description,
        type: "info",
        duration: 5000,
      });
    },
  };

  return {
    toaster: globalToaster,
    toast,
    iconMap,
  };
}
