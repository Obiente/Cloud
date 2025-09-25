export * from './configApi';
import { ConfigApi } from './configApi';
export * from './containerApi';
import { ContainerApi } from './containerApi';
export * from './distributionApi';
import { DistributionApi } from './distributionApi';
export * from './execApi';
import { ExecApi } from './execApi';
export * from './imageApi';
import { ImageApi } from './imageApi';
export * from './networkApi';
import { NetworkApi } from './networkApi';
export * from './nodeApi';
import { NodeApi } from './nodeApi';
export * from './pluginApi';
import { PluginApi } from './pluginApi';
export * from './secretApi';
import { SecretApi } from './secretApi';
export * from './serviceApi';
import { ServiceApi } from './serviceApi';
export * from './sessionApi';
import { SessionApi } from './sessionApi';
export * from './swarmApi';
import { SwarmApi } from './swarmApi';
export * from './systemApi';
import { SystemApi } from './systemApi';
export * from './taskApi';
import { TaskApi } from './taskApi';
export * from './volumeApi';
import { VolumeApi } from './volumeApi';
import * as http from 'http';

export class HttpError extends Error {
    constructor (public response: http.IncomingMessage, public body: any, public statusCode?: number) {
        super('HTTP request failed');
        this.name = 'HttpError';
    }
}

export { RequestFile } from '../model/models';

export const APIS = [ConfigApi, ContainerApi, DistributionApi, ExecApi, ImageApi, NetworkApi, NodeApi, PluginApi, SecretApi, ServiceApi, SessionApi, SwarmApi, SystemApi, TaskApi, VolumeApi];
