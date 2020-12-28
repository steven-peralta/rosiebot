import WaifuApi from './WaifuApi';
import config from '../config';

const MwlApi = new WaifuApi(config.waifuAPIKey);
export default MwlApi;
