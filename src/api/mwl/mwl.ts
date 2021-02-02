import MWLAPI from '$api/mwl/MWLAPI';
import config from '$config';

const mwl = new MWLAPI(config.waifuApiKey);

export default mwl;
