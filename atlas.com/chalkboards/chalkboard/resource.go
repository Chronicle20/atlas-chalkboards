package chalkboard

import (
	"atlas-chalkboards/character"
	"atlas-chalkboards/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		cr := router.PathPrefix("/chalkboards/{characterId}").Subrouter()
		cr.HandleFunc("", registerGet("chalkboard_by_character_id", handleGetChalkboard)).Methods(http.MethodGet)

		mr := router.PathPrefix("/worlds/{worldId}/channels/{channelId}/maps/{mapId}/chalkboards").Subrouter()
		mr.HandleFunc("", registerGet("chalkboards_in_map", handleGetChalkboardsInMap)).Methods(http.MethodGet)
	}
}

func handleGetChalkboard(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p, err := GetById(characterId)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(p))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleGetChalkboardsInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId byte) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					cip := character.InMapProvider(d.Context())(worldId, channelId, mapId)
					fcip := model.FilteredProvider(cip, model.Filters[uint32](func(cid uint32) bool {
						_, err := GetById(cid)
						return err == nil
					}))
					cimp := model.SliceMap[uint32, Model](GetById)(fcip)(model.ParallelMap())

					res, err := model.SliceMap(Transform)(cimp)(model.ParallelMap())()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
				}
			})
		})
	})
}
