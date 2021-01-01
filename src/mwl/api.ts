import WaifuApi from './WaifuApi';
import config from '../config';

const api = new WaifuApi(config.waifuApiKey);
export default api;
