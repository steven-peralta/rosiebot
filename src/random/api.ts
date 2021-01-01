import config from '../config';
import RandomApi from './RandomApi';

const api = new RandomApi(config.randomOrgApiKey);
export default api;
