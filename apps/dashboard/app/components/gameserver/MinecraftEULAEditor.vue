<template>
  <div class="h-full flex items-center justify-center p-8">
    <div class="w-full max-w-2xl">
      <OuiCard>
        <OuiCardHeader>
          <OuiFlex justify="between" align="center">
            <div>
              <OuiText as="h2" size="lg" weight="semibold">
                Minecraft EULA
              </OuiText>
              <OuiText size="sm" color="secondary" class="mt-1">
                End User License Agreement
              </OuiText>
            </div>
            <div
              v-if="eulaAccepted"
              class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-success/10"
            >
              <CheckCircleIcon class="h-5 w-5 text-success" />
              <OuiText size="sm" weight="medium" color="success">
                Accepted
              </OuiText>
            </div>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="md">
            <div
              class="p-4 rounded-lg bg-surface-elevated border border-border-default"
            >
              <OuiText size="sm" class="whitespace-pre-wrap leading-relaxed">
                {{ eulaText }}
              </OuiText>
            </div>

            <OuiFlex gap="sm" align="center" class="flex-wrap">
              <OuiButton
                v-if="!eulaAccepted"
                variant="solid"
                :loading="isSaving"
                @click="handleAcceptEULA"
              >
                <CheckCircleIcon class="h-4 w-4 mr-2" />
                I Accept the EULA
              </OuiButton>
              <OuiButton
                v-else
                variant="outline"
                @click="handleRejectEULA"
                :loading="isSaving"
              >
                <XCircleIcon class="h-4 w-4 mr-2" />
                Revoke Acceptance
              </OuiButton>
              <OuiButton
                variant="ghost"
                size="sm"
                @click="openEULALink"
              >
                <ArrowTopRightOnSquareIcon class="h-4 w-4 mr-2" />
                View Full EULA
              </OuiButton>
            </OuiFlex>

            <div class="pt-4 border-t border-border-default">
              <OuiText size="xs" color="muted">
                By accepting the EULA, you agree to Mojang's terms of service.
                The EULA file will be updated automatically.
              </OuiText>
            </div>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  CheckCircleIcon,
  XCircleIcon,
  ArrowTopRightOnSquareIcon,
} from "@heroicons/vue/24/outline";

interface Props {
  fileContent: string;
  isSaving?: boolean;
}

interface Emits {
  (e: "save", content: string): void;
}

const props = withDefaults(defineProps<Props>(), {
  isSaving: false,
});

const emit = defineEmits<Emits>();

const eulaText = `MINECRAFT END USER LICENSE AGREEMENT

By clicking "I Accept" or by installing, copying, or otherwise using Minecraft, you agree to be bound by the terms of this End User License Agreement ("EULA").

Please read this EULA carefully before accepting it. If you do not agree to the terms of this EULA, do not install or use Minecraft.

This EULA is a legal agreement between you (either an individual or a single entity) and Mojang AB ("Mojang") for the Minecraft software product, which includes computer software and may include associated media, printed materials, and "online" or electronic documentation ("Minecraft").

By installing, copying, or otherwise using Minecraft, you agree to be bound by the terms of this EULA. If you do not agree to the terms of this EULA, do not install or use Minecraft.

For the full EULA, please visit: https://account.mojang.com/documents/minecraft_eula`;

const eulaAccepted = computed(() => {
  return props.fileContent.includes("eula=true");
});

function handleAcceptEULA() {
  const newContent = `#By changing the setting below to TRUE you are indicating your agreement to our EULA (https://account.mojang.com/documents/minecraft_eula).
#${new Date().toISOString()}
eula=true
`;
  emit("save", newContent);
}

function handleRejectEULA() {
  const newContent = `#By changing the setting below to TRUE you are indicating your agreement to our EULA (https://account.mojang.com/documents/minecraft_eula).
#${new Date().toISOString()}
eula=false
`;
  emit("save", newContent);
}

function openEULALink() {
  window.open("https://account.mojang.com/documents/minecraft_eula", "_blank");
}
</script>

