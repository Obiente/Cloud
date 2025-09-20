#!/usr/bin/env node

import { execSync } from 'child_process';
import { existsSync, mkdirSync } from 'fs';
import { join } from 'path';

const PROTO_DIR = join(process.cwd(), 'proto');
const GENERATED_DIR = join(process.cwd(), 'generated');

// Ensure generated directory exists
if (!existsSync(GENERATED_DIR)) {
  mkdirSync(GENERATED_DIR, { recursive: true });
}

console.log('Generating TypeScript code from protobuf definitions...');

try {
  // Generate TypeScript code using protoc with connectrpc plugins
  const command = [
    'npx protoc',
    `--proto_path=${PROTO_DIR}`,
    '--es_out=generated',
    '--es_opt=target=ts',
    '--connect-es_out=generated',
    '--connect-es_opt=target=ts',
    `${PROTO_DIR}/obiente/cloud/**/*.proto`
  ].join(' ');

  console.log('Running command:', command);
  execSync(command, { stdio: 'inherit' });

  console.log('✅ Protocol buffer code generation completed successfully!');
} catch (error) {
  console.error('❌ Failed to generate protobuf code:', error.message);
  process.exit(1);
}