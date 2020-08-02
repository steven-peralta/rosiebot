import { IWaifuDocument, IWaifuModel } from './waifus.types';
import { Waifu, waifuApi } from '../../waifu/api';

export async function findOneOrCreate(
  this: IWaifuModel,
  waifu: Waifu
): Promise<IWaifuDocument> {
  const record = await this.findOne({ mwlId: waifu.id });
  if (record) {
    return record;
  } else {
    const appearsIn: string[] =
      waifu.appearances?.reduce((acc: string[], val) => {
        acc.push(val.name);
        return acc;
      }, []) ?? [];
    return this.create({
      mwlId: waifu.id,
      name: waifu.name,
      url: waifu.url,
      originalName: waifu.original_name,
      displayPictureURL: waifu.display_picture,
      description: waifu.description,
      weight: waifu.weight,
      height: waifu.height,
      bust: waifu.bust,
      hip: waifu.hip,
      waist: waifu.waist,
      bloodType: waifu.blood_type,
      origin: waifu.origin,
      age: waifu.age,
      birthdayMonth: waifu.birthday_month,
      birthdayDay: waifu.birthday_day,
      birthdayYear: waifu.birthday_year,
      husbando: waifu.husbando,
      appearsIn,
      series: waifu.series?.name,
    });
  }
}
