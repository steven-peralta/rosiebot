import WaifuApi from './WaifuApi';
import config from '../config';

const mwlApi = new WaifuApi(config.waifuApiKey);
export default mwlApi;
